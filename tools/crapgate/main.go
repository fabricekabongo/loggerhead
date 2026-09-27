package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type options struct {
	baseReport string
	headReport string
	baseRef    string
	headRef    string
}

type changedFile struct {
	status  string
	path    string
	oldPath string
}

type changedFunctionSet struct {
	base           map[functionID]string
	head           map[functionID]string
	renamedBaseIDs map[functionID]functionID
	testOnly       bool
}

type functionSymbol struct {
	packageName string
	receiver    string
	name        string
}

func main() {
	log.SetFlags(0)
	// Process termination cannot be reached by the in-process Go test harness.
	// skipcq: TCV-001
	if err := execute(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func execute(args []string) error {
	config, err := parseOptions(args)
	if err != nil {
		return err
	}
	return run(config, os.Stdout)
}

func parseOptions(args []string) (options, error) {
	var config options
	flags := flag.NewFlagSet("crapgate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&config.baseReport, "base-report", "", "trusted go-crap JSON report for base ref")
	flags.StringVar(&config.headReport, "head-report", "", "go-crap JSON report for head ref")
	flags.StringVar(&config.baseRef, "base-ref", "", "Git base ref used to classify function source changes")
	flags.StringVar(&config.headRef, "head-ref", "", "Git head ref used to classify function source changes")
	if err := flags.Parse(args); err != nil {
		return options{}, fmt.Errorf("parse arguments: %w", err)
	}
	if config.baseReport == "" || config.headReport == "" || config.baseRef == "" || config.headRef == "" {
		return options{}, fmt.Errorf("usage: crapgate --base-report FILE --head-report FILE --base-ref REF --head-ref REF")
	}
	return config, nil
}

func run(config options, output io.Writer) error {
	if err := validateRefs(config.baseRef, config.headRef); err != nil {
		return err
	}
	base, head, err := loadReports(config.baseReport, config.headReport)
	if err != nil {
		return err
	}
	changed, err := collectChangedFunctions(config.baseRef, config.headRef)
	if err != nil {
		return err
	}
	var results []result
	if changed.testOnly {
		results, err = evaluateTestOnlyCRAPChanges(base, head)
	} else {
		results, err = evaluateWithRenamedBaseIDs(base, head, changed.base, changed.head, changed.renamedBaseIDs)
		if err == nil {
			var testResults []result
			testResults, err = evaluateUnaffectedCRAPChanges(base, head, changed.base, changed.head, changed.renamedBaseIDs)
			results = append(results, testResults...)
		}
	}
	if err != nil {
		return err
	}
	if err := writeWorst(output, head, 10); err != nil {
		return err
	}
	return writeResults(output, results)
}

func writeWorst(output io.Writer, entries map[functionID]entry, limit int) error {
	type ranked struct {
		id   functionID
		item entry
	}
	all := make([]ranked, 0, len(entries))
	for id, item := range entries {
		all = append(all, ranked{id: id, item: item})
	}
	sort.Slice(all, func(i, j int) bool {
		if *all[i].item.CRAP == *all[j].item.CRAP {
			return all[i].id.String() < all[j].id.String()
		}
		return *all[i].item.CRAP > *all[j].item.CRAP
	})
	if limit > len(all) {
		limit = len(all)
	}
	for _, ranked := range all[:limit] {
		if _, err := fmt.Fprintf(output, "worst %s:%d CRAP=%.17g CC=%d statement-coverage=%.17g%%\n", ranked.id, *ranked.item.Line, *ranked.item.CRAP, *ranked.item.Cyclomatic, *ranked.item.Coverage); err != nil {
			return fmt.Errorf("write worst CRAP function: %w", err)
		}
	}
	return nil
}

func validateRefs(baseRef, headRef string) error {
	for _, ref := range []string{baseRef, headRef} {
		if _, err := gitOutput("rev-parse", "--verify", ref+"^{commit}"); err != nil {
			return fmt.Errorf("invalid Git ref %q: %w", ref, err)
		}
	}
	return nil
}

func loadReports(basePath, headPath string) (map[functionID]entry, map[functionID]entry, error) {
	base, err := readReport(basePath, "base")
	if err != nil {
		return nil, nil, err
	}
	head, err := readReport(headPath, "head")
	if err != nil {
		return nil, nil, err
	}
	return base, head, nil
}

func readReport(path, label string) (map[functionID]entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s report: %w", label, err)
	}
	entries, err := parseReport(data)
	if err != nil {
		return nil, fmt.Errorf("%s report: %w", label, err)
	}
	return entries, nil
}

func collectChangedFunctions(baseRef, headRef string) (changedFunctionSet, error) {
	files, err := changedFiles(baseRef, headRef)
	if err != nil {
		return changedFunctionSet{}, err
	}
	baseFunctions := make(map[functionID]string)
	headFunctions := make(map[functionID]string)
	renamedBaseIDs := make(map[functionID]functionID)
	testOnly := len(files) > 0
	for _, file := range files {
		if !isTestOnlyChange(file) {
			testOnly = false
		}
		if err := collectOneFile(file, baseRef, headRef, baseFunctions, headFunctions, renamedBaseIDs); err != nil {
			return changedFunctionSet{}, err
		}
	}
	if err := addMovedFunctionAliases(baseFunctions, headFunctions, renamedBaseIDs); err != nil {
		return changedFunctionSet{}, err
	}
	return changedFunctionSet{base: baseFunctions, head: headFunctions, renamedBaseIDs: renamedBaseIDs, testOnly: testOnly}, nil
}

func addMovedFunctionAliases(base, head map[functionID]string, aliases map[functionID]functionID) error {
	baseSymbols := indexFunctionSymbols(base)
	headSymbols := indexFunctionSymbols(head)
	for id := range head {
		if _, exists := base[id]; exists {
			continue
		}
		symbol := symbolFor(id)
		candidates := baseSymbols[symbol]
		if len(candidates) == 0 {
			continue
		}
		if len(candidates) != 1 || len(headSymbols[symbol]) != 1 {
			return fmt.Errorf("ambiguous moved function identity %s", id)
		}
		if _, exists := aliases[id]; !exists {
			aliases[id] = candidates[0]
		}
	}
	return nil
}

func indexFunctionSymbols(functions map[functionID]string) map[functionSymbol][]functionID {
	indexed := make(map[functionSymbol][]functionID)
	for id := range functions {
		symbol := symbolFor(id)
		indexed[symbol] = append(indexed[symbol], id)
	}
	return indexed
}

func symbolFor(id functionID) functionSymbol {
	return functionSymbol{packageName: id.Package, receiver: id.Receiver, name: id.Name}
}

func isTestOnlyChange(file changedFile) bool {
	return strings.HasSuffix(file.path, "_test.go") || strings.HasPrefix(file.path, "quality/go-crap-fixtures/")
}

func evaluateTestOnlyCRAPChanges(base, head map[functionID]entry) ([]result, error) {
	if len(base) != len(head) {
		return nil, fmt.Errorf("base and head CRAP reports contain different production function counts")
	}
	results := make([]result, 0)
	for id, baseEntry := range base {
		headEntry, ok := head[id]
		if !ok {
			return nil, fmt.Errorf("head CRAP report missing production function %s", id)
		}
		item, changed := testOnlyCRAPResult(id, baseEntry, headEntry)
		if changed {
			results = append(results, item)
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID.String() < results[j].ID.String() })
	return results, nil
}

func testOnlyCRAPResult(id functionID, base, head entry) (result, bool) {
	if *base.CRAP == *head.CRAP {
		return result{}, false
	}
	allowed := improvesOrMeetsLimit(*base.CRAP, *head.CRAP, maxCRAP)
	item := result{ID: id, Kind: "coverage", Base: *base.CRAP, Head: *head.CRAP, Delta: *head.CRAP - *base.CRAP, BaseCC: *base.Cyclomatic, HeadCC: *head.Cyclomatic, CCDelta: *head.Cyclomatic - *base.Cyclomatic, Allowed: allowed, Reason: "test-only changes may not worsen legacy CRAP or exceed 10"}
	return item, true
}

func changedFiles(baseRef, headRef string) ([]changedFile, error) {
	output, err := gitOutput("diff", "--find-renames", "--name-status", "--diff-filter=ACMRTD", baseRef+".."+headRef, "--", "*.go")
	if err != nil {
		return nil, fmt.Errorf("list changed Go files: %w", err)
	}
	files, err := parseChangedFiles(output)
	if err != nil {
		return nil, err
	}
	if err := rejectNestedModuleBoundaryChanges(baseRef, headRef); err != nil {
		return nil, err
	}
	return files, nil
}

func rejectNestedModuleBoundaryChanges(baseRef, headRef string) error {
	output, err := gitOutput("diff", "--find-renames", "--name-status", "--diff-filter=ACMRTD", baseRef+".."+headRef, "--", ":(glob)**/go.mod")
	if err != nil {
		return fmt.Errorf("list changed module files: %w", err)
	}
	files, err := parseChangedFiles(output)
	if err != nil {
		return err
	}
	for _, file := range files {
		changed, err := nestedModuleBoundaryChanged(baseRef, file)
		if err != nil {
			return err
		}
		if changed {
			return fmt.Errorf("nested module boundary changed at %s; Go source inventory cannot be assessed", moduleChangePath(file))
		}
	}
	return nil
}

func nestedModuleBoundaryChanged(baseRef string, file changedFile) (bool, error) {
	if file.status != "A" && file.status != "D" && file.status != "R" {
		return false, nil
	}
	paths := []string{file.path}
	if file.status == "R" {
		paths = append(paths, file.oldPath)
	}
	for _, path := range paths {
		if !isNestedModulePath(path) {
			continue
		}
		containsGo, err := moduleContainsGoFile(baseRef, filepath.ToSlash(filepath.Dir(path)))
		if err != nil || containsGo {
			return containsGo, err
		}
	}
	return false, nil
}

func moduleContainsGoFile(ref, directory string) (bool, error) {
	listing, err := gitOutput("ls-tree", "-r", "--name-only", ref, "--", directory)
	if err != nil {
		return false, fmt.Errorf("list Go sources under nested module %s at %s: %w", directory, ref, err)
	}
	for _, path := range strings.Split(listing, "\n") {
		if strings.HasSuffix(path, ".go") {
			return true, nil
		}
	}
	return false, nil
}

func moduleChangePath(file changedFile) string {
	if file.status == "R" {
		return file.oldPath + " -> " + file.path
	}
	return file.path
}

func isNestedModulePath(path string) bool {
	return strings.Contains(path, "/")
}

func collectOneFile(file changedFile, baseRef, headRef string, baseFunctions, headFunctions map[functionID]string, renamedBaseIDs map[functionID]functionID) error {
	switch file.status {
	case "R":
		return collectRenamedFile(file, baseRef, headRef, baseFunctions, headFunctions, renamedBaseIDs)
	case "D":
		return collectDeletedFile(file, baseRef)
	default:
		return collectModifiedFile(file, baseRef, headRef, baseFunctions, headFunctions)
	}
}

func collectRenamedFile(file changedFile, baseRef, headRef string, baseFunctions, headFunctions map[functionID]string, aliases map[functionID]functionID) error {
	baseIncluded, err := collectRenamedBase(file, baseRef, baseFunctions, aliases)
	if err != nil {
		return err
	}
	headIncluded, err := collectRenamedHead(file, headRef, headFunctions, baseIncluded)
	if err != nil {
		return err
	}
	if baseIncluded && !headIncluded {
		return fmt.Errorf("production Go file rename from %s to out-of-scope path %s cannot be assessed", file.oldPath, file.path)
	}
	return nil
}

func collectRenamedBase(file changedFile, baseRef string, functions map[functionID]string, aliases map[functionID]functionID) (bool, error) {
	if !isProductionGoPath(file.oldPath) {
		return false, nil
	}
	return collectRenamedSource(functions, baseRef, file.oldPath, file.path, aliases)
}

func collectRenamedHead(file changedFile, headRef string, functions map[functionID]string, baseIncluded bool) (bool, error) {
	if !isProductionGoPath(file.path) {
		if baseIncluded {
			return false, fmt.Errorf("production Go file rename from %s to excluded path %s cannot be assessed", file.oldPath, file.path)
		}
		return false, nil
	}
	return collectProductionFile(functions, headRef, file.path)
}

func collectDeletedFile(file changedFile, baseRef string) error {
	if !isProductionGoPath(file.path) {
		return nil
	}
	included, err := pathInScannedModule(baseRef, file.path)
	if err != nil || !included {
		return err
	}
	inBuild, err := deletedFileInBuild(baseRef, file.path)
	if err != nil || !inBuild {
		return err
	}
	return fmt.Errorf("deleted production Go file %s cannot be assessed against a head CRAP report", file.path)
}

func deletedFileInBuild(ref, path string) (bool, error) {
	return sourceInLinuxDefaultBuild(ref, path, "deleted source")
}

func collectModifiedFile(file changedFile, baseRef, headRef string, baseFunctions, headFunctions map[functionID]string) error {
	if !isProductionGoPath(file.path) {
		return nil
	}
	baseIncluded := false
	if file.status != "A" {
		included, err := collectProductionFile(baseFunctions, baseRef, file.path)
		if err != nil {
			return err
		}
		baseIncluded = included
	}
	headIncluded, err := collectProductionFile(headFunctions, headRef, file.path)
	if err != nil {
		return err
	}
	if baseIncluded && !headIncluded {
		return fmt.Errorf("production Go file %s left the default build and cannot be assessed", file.path)
	}
	return nil
}

func collectProductionFile(functions map[functionID]string, ref, path string) (bool, error) {
	included, err := pathInScannedModule(ref, path)
	if err != nil || !included {
		return included, err
	}
	inBuild, err := sourceInLinuxDefaultBuild(ref, path, "source")
	if err != nil || !inBuild {
		return false, err
	}
	return true, addFileFunctions(functions, ref, path)
}

func collectRenamedSource(functions map[functionID]string, ref, sourcePath, identityPath string, aliases map[functionID]functionID) (bool, error) {
	included, err := pathInScannedModule(ref, sourcePath)
	if err != nil || !included {
		return included, err
	}
	inBuild, err := sourceInLinuxDefaultBuild(ref, sourcePath, "source")
	if err != nil || !inBuild {
		return false, err
	}
	return true, addFileFunctionsAt(functions, ref, sourcePath, identityPath, aliases)
}

func sourceInLinuxDefaultBuild(ref, path, label string) (bool, error) {
	source, err := gitOutput("show", ref+":"+path)
	if err != nil {
		return false, fmt.Errorf("read %s %s:%s: %w", label, ref, path, err)
	}
	inBuild, err := fileInLinuxDefaultBuild(path, []byte(source))
	if err != nil {
		return false, fmt.Errorf("check build constraints for %s %s at %s: %w", label, path, ref, err)
	}
	return inBuild, nil
}

func isProductionGoPath(path string) bool {
	return !strings.HasSuffix(path, "_test.go") && !pathExcludedByGoPattern(path)
}

func pathExcludedByGoPattern(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for index, part := range parts[:len(parts)-1] {
		vendorDescendant := part == "vendor" && len(parts)-index > 2
		if vendorDescendant || part == "testdata" || strings.HasPrefix(part, ".") || strings.HasPrefix(part, "_") {
			return true
		}
	}
	return false
}

func pathInScannedModule(ref, path string) (bool, error) {
	directory := filepath.ToSlash(filepath.Dir(path))
	for directory != "." {
		modPath := directory + "/go.mod"
		listing, err := gitOutput("ls-tree", "--name-only", ref, "--", modPath)
		if err != nil {
			return false, fmt.Errorf("check nested module %s at %s: %w", modPath, ref, err)
		}
		if strings.TrimSpace(listing) == modPath {
			return false, nil
		}
		directory = filepath.ToSlash(filepath.Dir(directory))
	}
	return true, nil
}

func parseChangedFiles(output string) ([]changedFile, error) {
	var files []changedFile
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) == 2 && fields[1] != "" {
			files = append(files, changedFile{status: fields[0], path: fields[1]})
			continue
		}
		if len(fields) == 3 && strings.HasPrefix(fields[0], "R") && fields[1] != "" && fields[2] != "" {
			files = append(files, changedFile{status: "R", oldPath: fields[1], path: fields[2]})
			continue
		}
		return nil, fmt.Errorf("invalid git diff name-status line %q", line)
	}
	return files, nil
}

func addFileFunctions(dst map[functionID]string, ref, path string) error {
	return addFileFunctionsAt(dst, ref, path, path, nil)
}

func addFileFunctionsAt(dst map[functionID]string, ref, sourcePath, identityPath string, aliases map[functionID]functionID) error {
	source, err := gitOutput("show", ref+":"+sourcePath)
	if err != nil {
		return fmt.Errorf("read %s:%s: %w", ref, sourcePath, err)
	}
	inBuild, err := fileInLinuxDefaultBuild(sourcePath, []byte(source))
	if err != nil {
		return fmt.Errorf("check build constraints for %s at %s: %w", sourcePath, ref, err)
	}
	if !inBuild {
		return nil
	}
	functions, err := parseFunctions(identityPath, "", []byte(source))
	if err != nil {
		return fmt.Errorf("parse %s at %s: %w", sourcePath, ref, err)
	}
	for id, declaration := range functions {
		dst[id] = declaration
		if sourcePath != identityPath && aliases != nil {
			baseID := id
			baseID.File = normalizeReportedPath(sourcePath)
			aliases[id] = baseID
		}
	}
	return nil
}

func fileInLinuxDefaultBuild(path string, source []byte) (bool, error) {
	context := build.Default
	context.GOOS = "linux"
	context.GOARCH = "amd64"
	context.CgoEnabled = true
	context.OpenFile = func(string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(source)), nil
	}
	return context.MatchFile(filepath.Dir(path), filepath.Base(path))
}

func writeResults(output io.Writer, results []result) error {
	failed := false
	for _, item := range results {
		if err := writeResult(output, item); err != nil {
			return fmt.Errorf("write CRAP policy result: %w", err)
		}
		failed = failed || !item.Allowed
	}
	if failed {
		return fmt.Errorf("CRAP policy failed")
	}
	return nil
}

func writeResult(output io.Writer, item result) error {
	if item.Kind == "new" {
		_, err := fmt.Fprintf(output, "%s %s CRAP=%.17g delta=+%.17g CC=%d allowed=%t (%s)\n", item.Kind, item.ID, item.Head, item.Delta, item.HeadCC, item.Allowed, item.Reason)
		return err
	}
	_, err := fmt.Fprintf(output, "%s %s CRAP base=%.17g head=%.17g delta=%+.17g CC base=%d head=%d delta=%+d allowed=%t (%s)\n", item.Kind, item.ID, item.Base, item.Head, item.Delta, item.BaseCC, item.HeadCC, item.CCDelta, item.Allowed, item.Reason)
	return err
}

func gitOutput(args ...string) (string, error) {
	command := exec.Command("git", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(output)), err)
	}
	return strings.ReplaceAll(string(output), "\r\n", "\n"), nil
}

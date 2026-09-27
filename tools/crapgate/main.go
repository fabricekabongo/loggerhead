package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	status string
	path   string
}

func main() {
	if err := execute(os.Args[1:]); err != nil {
		fatalf("%v", err)
	}
}

func execute(args []string) error {
	config, err := parseOptions(args)
	if err != nil {
		return err
	}
	return run(config)
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

func run(config options) error {
	if err := validateRefs(config.baseRef, config.headRef); err != nil {
		return err
	}
	base, head, err := loadReports(config.baseReport, config.headReport)
	if err != nil {
		return err
	}
	baseFunctions, headFunctions, testOnly, err := collectChangedFunctions(config.baseRef, config.headRef)
	if err != nil {
		return err
	}
	var results []result
	if testOnly {
		results, err = evaluateTestOnlyCRAPChanges(base, head)
	} else {
		results, err = evaluate(base, head, baseFunctions, headFunctions)
	}
	if err != nil {
		return err
	}
	if err := writeWorst(os.Stdout, head, 10); err != nil {
		return err
	}
	return writeResults(os.Stdout, results)
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

func collectChangedFunctions(baseRef, headRef string) (map[functionID]string, map[functionID]string, bool, error) {
	files, err := changedFiles(baseRef, headRef)
	if err != nil {
		return nil, nil, false, err
	}
	baseFunctions := make(map[functionID]string)
	headFunctions := make(map[functionID]string)
	testOnly := len(files) > 0
	for _, file := range files {
		if !strings.HasSuffix(file.path, "_test.go") && !strings.HasPrefix(file.path, "quality/go-crap-fixtures/") {
			testOnly = false
		}
		if err := collectOneFile(file, baseRef, headRef, baseFunctions, headFunctions); err != nil {
			return nil, nil, false, err
		}
	}
	return baseFunctions, headFunctions, testOnly, nil
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
	output, err := gitOutput("diff", "--no-renames", "--name-status", "--diff-filter=ACMRT", baseRef+"..."+headRef, "--", "*.go")
	if err != nil {
		return nil, fmt.Errorf("list changed Go files: %w", err)
	}
	return parseChangedFiles(output)
}

func collectOneFile(file changedFile, baseRef, headRef string, baseFunctions, headFunctions map[functionID]string) error {
	// The golden fixture is a separate Go module and is not production code in ./....
	if strings.HasSuffix(file.path, "_test.go") || strings.HasPrefix(file.path, "quality/go-crap-fixtures/") {
		return nil
	}
	if file.status != "A" {
		if err := addFileFunctions(baseFunctions, baseRef, file.path); err != nil {
			return err
		}
	}
	return addFileFunctions(headFunctions, headRef, file.path)
}

func parseChangedFiles(output string) ([]changedFile, error) {
	var files []changedFile
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) != 2 || fields[1] == "" {
			return nil, fmt.Errorf("invalid git diff name-status line %q", line)
		}
		files = append(files, changedFile{status: fields[0], path: fields[1]})
	}
	return files, nil
}

func addFileFunctions(dst map[functionID]string, ref, path string) error {
	source, err := gitOutput("show", ref+":"+path)
	if err != nil {
		return fmt.Errorf("read %s:%s: %w", ref, path, err)
	}
	functions, err := parseFunctions(path, "", []byte(source))
	if err != nil {
		return fmt.Errorf("parse %s at %s: %w", path, ref, err)
	}
	for id, declaration := range functions {
		dst[id] = declaration
	}
	return nil
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

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

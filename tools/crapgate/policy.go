package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"
)

const maxCRAP = 10.0
const maxCyclomatic = 10
const reportSchemaVersion = "1.1.0"

type report struct {
	Version string `json:"version"`
	Summary struct {
		TotalFunctions *int `json:"total_funcs"`
	} `json:"summary"`
	Entries []entry `json:"entries"`
}

type entry struct {
	File       string   `json:"file"`
	Package    string   `json:"package"`
	Function   string   `json:"function"`
	Receiver   string   `json:"receiver"`
	CRAP       *float64 `json:"crap"`
	Cyclomatic *int     `json:"cyclomatic"`
	Coverage   *float64 `json:"coverage"`
	Line       *int     `json:"line"`
}

type functionID struct {
	Package  string
	File     string
	Receiver string
	Name     string
}

func (id functionID) String() string {
	name := id.Name
	if id.Receiver != "" {
		name = id.Receiver + "." + name
	}
	return id.Package + "/" + id.File + ":" + name
}

type result struct {
	ID      functionID
	Kind    string
	Base    float64
	Head    float64
	Delta   float64
	BaseCC  int
	HeadCC  int
	CCDelta int
	Allowed bool
	Reason  string
}

func parseReport(data []byte) (map[functionID]entry, error) {
	var parsed report
	if err := decodeReport(data, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Entries) == 0 {
		return nil, fmt.Errorf("CRAP report has no function entries")
	}
	if parsed.Summary.TotalFunctions == nil || *parsed.Summary.TotalFunctions != len(parsed.Entries) {
		return nil, fmt.Errorf("CRAP report summary total_funcs does not match %d entries", len(parsed.Entries))
	}
	entries := make(map[functionID]entry, len(parsed.Entries))
	for i, item := range parsed.Entries {
		id, err := validateEntry(item)
		if err != nil {
			return nil, fmt.Errorf("CRAP report entry %d: %w", i, err)
		}
		if _, exists := entries[id]; exists {
			return nil, fmt.Errorf("duplicate CRAP report entry for %s", id)
		}
		entries[id] = item
	}
	return entries, nil
}

func decodeReport(data []byte, parsed *report) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(parsed); err != nil {
		return fmt.Errorf("decode CRAP report: %w", err)
	}
	if parsed.Version != reportSchemaVersion {
		return fmt.Errorf("CRAP report schema version %q, want %q", parsed.Version, reportSchemaVersion)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("decode CRAP report: trailing JSON data")
	}
	return nil
}

func validateEntry(item entry) (functionID, error) {
	id, err := idFromEntry(item)
	if err != nil {
		return functionID{}, err
	}
	if err := validateMetrics(item); err != nil {
		return functionID{}, err
	}
	return id, nil
}

func validateMetrics(item entry) error {
	if item.Coverage == nil || item.Line == nil || item.Cyclomatic == nil || item.CRAP == nil {
		return fmt.Errorf("missing coverage, line, cyclomatic, or crap")
	}
	if err := validateCoverage(*item.Coverage); err != nil {
		return err
	}
	if err := validateComplexity(*item.Line, *item.Cyclomatic); err != nil {
		return err
	}
	return validateCRAPScore(*item.CRAP)
}

func validateCoverage(coverage float64) error {
	if math.IsNaN(coverage) || math.IsInf(coverage, 0) || coverage < 0 || coverage > 100 {
		return fmt.Errorf("invalid coverage %v", coverage)
	}
	return nil
}

func validateComplexity(line, complexity int) error {
	if line <= 0 || complexity <= 0 {
		return fmt.Errorf("line and cyclomatic complexity must be positive")
	}
	return nil
}

func validateCRAPScore(score float64) error {
	if math.IsNaN(score) || math.IsInf(score, 0) || score <= 0 {
		return fmt.Errorf("invalid CRAP score %v", score)
	}
	return nil
}

func idFromEntry(item entry) (functionID, error) {
	file := normalizeReportedPath(item.File)
	if file == "" || item.Package == "" || item.Function == "" {
		return functionID{}, fmt.Errorf("missing file, package, or function identity")
	}
	receiver := strings.TrimSpace(item.Receiver)
	name := item.Function
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		reportedReceiver := name[:dot]
		name = name[dot+1:]
		if receiver == "" || reportedReceiver != receiver {
			return functionID{}, fmt.Errorf("method %q has inconsistent receiver %q", item.Function, receiver)
		}
	} else if receiver != "" {
		return functionID{}, fmt.Errorf("function %q has receiver %q without method identity", item.Function, receiver)
	}
	if name == "" {
		return functionID{}, fmt.Errorf("empty function name")
	}
	return functionID{Package: item.Package, File: file, Receiver: receiver, Name: name}, nil
}

func normalizeReportedPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.TrimPrefix(path, "./")
	for strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
	}
	return strings.TrimPrefix(path, "/")
}

func parseFunctions(filename, packageName string, source []byte) (map[functionID]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	if packageName == "" {
		packageName = file.Name.Name
	}
	functions := make(map[functionID]string)
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		var receiver string
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			receiver, err = formatNode(fset, fn.Recv.List[0].Type)
			if err != nil {
				return nil, fmt.Errorf("format receiver in %s: %w", filename, err)
			}
		}
		id := functionID{Package: packageName, File: normalizeReportedPath(filename), Receiver: receiver, Name: fn.Name.Name}
		text, err := formatNode(fset, fn)
		if err != nil {
			return nil, fmt.Errorf("format %s in %s: %w", fn.Name.Name, filename, err)
		}
		functions[id] = text
	}
	return functions, nil
}

func formatNode(fset *token.FileSet, node ast.Node) (string, error) {
	var output bytes.Buffer
	if err := format.Node(&output, fset, node); err != nil {
		return "", err
	}
	return output.String(), nil
}

func evaluate(base, head map[functionID]entry, baseFunctions, headFunctions map[functionID]string) ([]result, error) {
	var results []result
	for id, body := range headFunctions {
		headEntry, found := findEntry(head, id)
		if !found {
			return nil, fmt.Errorf("head CRAP report missing function %s", id)
		}
		baseBody, existed := baseFunctions[id]
		if !existed {
			allowed := *headEntry.CRAP <= maxCRAP && *headEntry.Cyclomatic <= maxCyclomatic
			reason := "new function must have CRAP <= 10 and cyclomatic complexity <= 10"
			results = append(results, result{ID: id, Kind: "new", Head: *headEntry.CRAP, Delta: *headEntry.CRAP, HeadCC: *headEntry.Cyclomatic, Allowed: allowed, Reason: reason})
			continue
		}
		if body == baseBody {
			continue
		}
		baseEntry, found := findEntry(base, id)
		if !found {
			return nil, fmt.Errorf("base CRAP report missing changed function %s", id)
		}
		crapAllowed := improvesOrMeetsLimit(*baseEntry.CRAP, *headEntry.CRAP, maxCRAP)
		complexityAllowed := improvesOrMeetsLimit(float64(*baseEntry.Cyclomatic), float64(*headEntry.Cyclomatic), float64(maxCyclomatic))
		allowed := crapAllowed && complexityAllowed
		reason := "changed function must satisfy CRAP and cyclomatic complexity limits or strictly improve legacy scores"
		results = append(results, result{ID: id, Kind: "changed", Base: *baseEntry.CRAP, Head: *headEntry.CRAP, Delta: *headEntry.CRAP - *baseEntry.CRAP, BaseCC: *baseEntry.Cyclomatic, HeadCC: *headEntry.Cyclomatic, CCDelta: *headEntry.Cyclomatic - *baseEntry.Cyclomatic, Allowed: allowed, Reason: reason})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID.String() < results[j].ID.String() })
	return results, nil
}

func improvesOrMeetsLimit(base, head, limit float64) bool {
	if base <= limit {
		return head <= limit
	}
	return head < base
}

func findEntry(entries map[functionID]entry, id functionID) (entry, bool) {
	if exact, ok := entries[id]; ok {
		return exact, true
	}
	return findPathMatch(entries, id)
}

func findPathMatch(entries map[functionID]entry, id functionID) (entry, bool) {
	var match entry
	count := 0
	for candidate, item := range entries {
		if candidate.Package != id.Package || candidate.Receiver != id.Receiver || candidate.Name != id.Name {
			continue
		}
		if candidate.File == id.File || strings.HasSuffix(id.File, "/"+candidate.File) || strings.HasSuffix(candidate.File, "/"+id.File) {
			match = item
			count++
		}
	}
	return match, count == 1
}

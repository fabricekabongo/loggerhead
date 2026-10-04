package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func testID(file, receiver, name string) functionID {
	return functionID{Package: "fixture", File: file, Receiver: receiver, Name: name}
}

func testEntry(id functionID, function string, score float64) entry {
	return entry{File: id.File, Package: id.Package, Receiver: id.Receiver, Function: function, CRAP: floatPtr(score), Cyclomatic: intPtr(1), Coverage: floatPtr(0), Line: intPtr(1)}
}

func floatPtr(value float64) *float64 { return &value }
func intPtr(value int) *int           { return &value }

func testEntryCC(id functionID, function string, score float64, cc int) entry {
	item := testEntry(id, function, score)
	item.Cyclomatic = intPtr(cc)
	return item
}

func testReport(entries ...entry) report {
	value := report{Version: reportSchemaVersion, Entries: entries}
	value.Summary.TotalFunctions = intPtr(len(entries))
	return value
}

func evaluateOne(t *testing.T, id functionID, baseScore, headScore float64, existed bool) result {
	return evaluateOneWithCC(t, id, baseScore, headScore, 1, 1, existed)
}

func evaluateOneWithCC(t *testing.T, id functionID, baseScore, headScore float64, baseCC, headCC int, existed bool) result {
	t.Helper()
	base := map[functionID]entry{}
	baseFunctions := map[functionID]string{}
	if existed {
		base[id] = testEntryCC(id, qualified(id), baseScore, baseCC)
		baseFunctions[id] = "func body A"
	}
	head := map[functionID]entry{id: testEntryCC(id, qualified(id), headScore, headCC)}
	return evaluateResult(t, base, head, baseFunctions, map[functionID]string{id: "func body B"})
}

func qualified(id functionID) string {
	if id.Receiver != "" {
		return id.Receiver + "." + id.Name
	}
	return id.Name
}

func evaluateResult(t *testing.T, base, head map[functionID]entry, baseFunctions, headFunctions map[functionID]string) result {
	t.Helper()
	results, err := evaluate(base, head, baseFunctions, headFunctions)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	return results[0]
}

func TestNewAndChangedThresholdIsExact(t *testing.T) {
	id := testID("pkg/file.go", "", "Run")
	for _, tc := range []struct {
		name    string
		base    float64
		head    float64
		existed bool
		want    bool
	}{
		{name: "new exactly ten", head: 10, want: true},
		{name: "new just over ten", head: 10.0000001, want: false},
		{name: "changed exactly ten remains ten", base: 10, head: 10, existed: true, want: true},
		{name: "changed at ten over threshold", base: 10, head: 10.0000001, existed: true, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateOne(t, id, tc.base, tc.head, tc.existed)
			if got.Allowed != tc.want {
				t.Fatalf("allowed=%t, want %t", got.Allowed, tc.want)
			}
		})
	}
}

func TestChangedLegacyHighScoreMustStrictlyImprove(t *testing.T) {
	id := testID("pkg/file.go", "", "Run")
	for _, tc := range []struct {
		name string
		head float64
		want bool
	}{
		{name: "equal", head: 15, want: false},
		{name: "reduced", head: 14.999, want: true},
		{name: "worse", head: 15.001, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateOne(t, id, 15, tc.head, true)
			if got.Allowed != tc.want || got.Delta != tc.head-15 {
				t.Fatalf("result = %#v", got)
			}
		})
	}
}

func TestMalformedAndMissingReportEntriesFailClosed(t *testing.T) {
	if _, err := parseReport([]byte("{")); err == nil {
		t.Fatal("malformed JSON accepted")
	}
	if _, err := parseReport([]byte(`{"version":"1.1.0","summary":{"total_funcs":0},"entries":[]}`)); err == nil {
		t.Fatal("empty function report accepted")
	}
	if _, err := parseReport([]byte(`{"version":"1.2.0","entries":[{}]}`)); err == nil {
		t.Fatal("wrong schema version accepted")
	}
	for _, field := range []string{"coverage", "line", "cyclomatic", "crap"} {
		t.Run("missing "+field, func(t *testing.T) {
			item := `{"file":"x.go","package":"fixture","function":"Run","crap":1,"coverage":0,"line":1,"cyclomatic":1}`
			value := map[string]string{"coverage": "0", "line": "1", "cyclomatic": "1", "crap": "1"}[field]
			item = strings.Replace(item, `,"`+field+`":`+value, "", 1)
			payload := `{"version":"1.1.0","summary":{"total_funcs":1},"entries":[` + item + `]}`
			if _, err := parseReport([]byte(payload)); err == nil {
				t.Fatalf("missing %s accepted", field)
			}
		})
	}
}

func TestInvalidReportMetricsFailClosed(t *testing.T) {
	for _, tc := range []struct {
		field string
		value string
	}{
		{field: "coverage", value: "100.01"},
		{field: "line", value: "0"},
		{field: "cyclomatic", value: "0"},
		{field: "crap", value: "0"},
	} {
		t.Run("invalid "+tc.field, func(t *testing.T) {
			item := `{"file":"x.go","package":"fixture","function":"Run","crap":1,"coverage":0,"line":1,"cyclomatic":1}`
			previous := map[string]string{"coverage": "0", "line": "1", "cyclomatic": "1", "crap": "1"}[tc.field]
			item = strings.Replace(item, `"`+tc.field+`":`+previous, `"`+tc.field+`":`+tc.value, 1)
			payload := `{"version":"1.1.0","summary":{"total_funcs":1},"entries":[` + item + `]}`
			if _, err := parseReport([]byte(payload)); err == nil {
				t.Fatalf("invalid %s value accepted", tc.field)
			}
		})
	}
}

func TestMissingReportEntriesFailClosed(t *testing.T) {
	id := testID("pkg/file.go", "", "Run")
	base := map[functionID]entry{id: testEntry(id, "Run", 15)}
	_, err := evaluate(base, map[functionID]entry{}, map[functionID]string{id: "before"}, map[functionID]string{id: "after"})
	if err == nil || !strings.Contains(err.Error(), "head CRAP report missing") {
		t.Fatalf("missing entry error = %v", err)
	}
	_, err = evaluate(map[functionID]entry{}, map[functionID]entry{id: testEntry(id, "Run", 5)}, map[functionID]string{id: "before"}, map[functionID]string{id: "after"})
	if err == nil || !strings.Contains(err.Error(), "base CRAP report missing") {
		t.Fatalf("missing base entry error = %v", err)
	}
}

func TestCyclomaticComplexityPolicy(t *testing.T) {
	id := testID("pkg/file.go", "", "Run")
	for _, tc := range []struct {
		name    string
		base    int
		head    int
		existed bool
		want    bool
	}{
		{name: "new at ten", head: 10, want: true},
		{name: "new over ten", head: 11, want: false},
		{name: "changed at ten remains ten", base: 10, head: 10, existed: true, want: true},
		{name: "changed at ten exceeds", base: 10, head: 11, existed: true, want: false},
		{name: "legacy equal", base: 11, head: 11, existed: true, want: false},
		{name: "legacy improves", base: 11, head: 10, existed: true, want: true},
		{name: "legacy worsens", base: 11, head: 12, existed: true, want: false},
		{name: "legacy score improves but complexity worsens", base: 11, head: 12, existed: true, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseCRAP, headCRAP := 5.0, 5.0
			if tc.name == "legacy score improves but complexity worsens" {
				baseCRAP, headCRAP = 15, 14
			}
			got := evaluateOneWithCC(t, id, baseCRAP, headCRAP, tc.base, tc.head, tc.existed)
			if got.Allowed != tc.want {
				t.Fatalf("allowed=%t, want %t", got.Allowed, tc.want)
			}
		})
	}
}

func TestDuplicateReportIdentityFails(t *testing.T) {
	data, err := json.Marshal(testReport(
		entry{File: "pkg/a.go", Package: "fixture", Function: "Run", CRAP: floatPtr(1), Cyclomatic: intPtr(1), Coverage: floatPtr(0), Line: intPtr(1)},
		entry{File: "../pkg/a.go", Package: "fixture", Function: "Run", CRAP: floatPtr(2), Cyclomatic: intPtr(2), Coverage: floatPtr(0), Line: intPtr(2)},
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseReport(data); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate identity error = %v", err)
	}
}

func TestPartialReportAndContradictoryMethodIdentityFail(t *testing.T) {
	item := entry{File: "pkg/a.go", Package: "fixture", Function: "*Counter.Run", Receiver: "*Counter", CRAP: floatPtr(2), Cyclomatic: intPtr(1), Coverage: floatPtr(0), Line: intPtr(1)}
	partial := testReport(item)
	partial.Summary.TotalFunctions = intPtr(2)
	data, err := json.Marshal(partial)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseReport(data); err == nil || !strings.Contains(err.Error(), "total_funcs") {
		t.Fatalf("partial report error = %v", err)
	}
	item.Receiver = "*Other"
	data, err = json.Marshal(testReport(item))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseReport(data); err == nil || !strings.Contains(err.Error(), "inconsistent receiver") {
		t.Fatalf("contradictory method identity error = %v", err)
	}
}

func TestUnchangedFunctionIsNotGated(t *testing.T) {
	id := testID("pkg/file.go", "", "Run")
	base := map[functionID]entry{id: testEntry(id, "Run", 99)}
	head := map[functionID]entry{id: testEntry(id, "Run", 100)}
	results, err := evaluate(base, head, map[functionID]string{id: "same declaration"}, map[functionID]string{id: "same declaration"})
	if err != nil || len(results) != 0 {
		t.Fatalf("unchanged function result = %#v, err=%v", results, err)
	}
}

func TestRenameAliasAppliesOnlyToBaseReportLookup(t *testing.T) {
	id := testID("pkg/new.go", "", "Run")
	oldID := testID("pkg/old.go", "", "Run")
	wrongHeadID := testID("pkg/old.go", "", "Run")
	_, err := evaluateWithRenamedBaseIDs(
		map[functionID]entry{oldID: testEntry(oldID, "Run", 15)},
		map[functionID]entry{wrongHeadID: testEntry(wrongHeadID, "Run", 14)},
		map[functionID]string{id: "func Run() { return 1 }"},
		map[functionID]string{id: "func Run() { return 2 }"},
		map[functionID]functionID{id: oldID},
	)
	if err == nil || !strings.Contains(err.Error(), "head CRAP report missing function") {
		t.Fatalf("renamed head report lookup error = %v", err)
	}
}

func TestReportIdentityIncludesPathAndReceiver(t *testing.T) {
	data, err := json.Marshal(testReport(
		entry{File: "../pkg/a.go", Package: "fixture", Function: "*Counter.Run", Receiver: "*Counter", CRAP: floatPtr(2), Cyclomatic: intPtr(1), Coverage: floatPtr(0), Line: intPtr(1)},
		entry{File: "pkg/b.go", Package: "fixture", Function: "Counter.Run", Receiver: "Counter", CRAP: floatPtr(3), Cyclomatic: intPtr(1), Coverage: floatPtr(0), Line: intPtr(1)},
	))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parseReport(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := entries[testID("pkg/a.go", "*Counter", "Run")]; !ok {
		t.Fatalf("pointer receiver/path identity missing: %#v", entries)
	}
	if _, ok := entries[testID("pkg/b.go", "Counter", "Run")]; !ok {
		t.Fatalf("value receiver/path identity missing: %#v", entries)
	}
}

func TestFindEntryMatchesUniqueSuffixAndRejectsAmbiguity(t *testing.T) {
	id := testID("repo/pkg/a.go", "", "Run")
	item := testEntry(testID("pkg/a.go", "", "Run"), "Run", 1)
	if found, ok := findEntry(map[functionID]entry{{Package: "fixture", File: "pkg/a.go", Name: "Run"}: item}, id); !ok || *found.CRAP != 1 {
		t.Fatalf("suffix match = %#v, %t", found, ok)
	}
	ambiguous := map[functionID]entry{
		{Package: "fixture", File: "pkg/a.go", Name: "Run"}:            item,
		{Package: "fixture", File: "other/repo/pkg/a.go", Name: "Run"}: item,
	}
	if _, ok := findEntry(ambiguous, id); ok {
		t.Fatal("ambiguous suffix match accepted")
	}
	if _, ok := findEntry(map[functionID]entry{}, id); ok {
		t.Fatal("missing path match accepted")
	}
}

func TestParseFunctionsUsesFullDeclarationAndReceiver(t *testing.T) {
	functions, err := parseFunctions("pkg/file.go", "fixture", []byte("package fixture\ntype Counter struct{}\nfunc (c *Counter) Run(x int) int { if x > 0 { return x }; return 0 }\n"))
	if err != nil {
		t.Fatal(err)
	}
	id := testID("pkg/file.go", "*Counter", "Run")
	body, ok := functions[id]
	if !ok || !strings.Contains(body, "Run(x int)") {
		t.Fatalf("function identity/source = %v, %q", ok, body)
	}
}

func TestParseFunctionsIgnoresDocumentationButTracksBodyChanges(t *testing.T) {
	const original = "package fixture\nfunc Run(value int) int { return value + 1 }\n"
	const documented = "package fixture\n// Run explains the value transformation.\nfunc Run(value int) int { return value + 1 }\n"
	const changedBody = "package fixture\n// Run explains the value transformation.\nfunc Run(value int) int { return value + 2 }\n"

	parse := func(source string) string {
		t.Helper()
		functions, err := parseFunctions("pkg/file.go", "fixture", []byte(source))
		if err != nil {
			t.Fatalf("parse function source: %v", err)
		}
		id := testID("pkg/file.go", "", "Run")
		declaration, ok := functions[id]
		if !ok {
			t.Fatalf("function fingerprint missing for %s", id)
		}
		return declaration
	}

	originalFingerprint := parse(original)
	if documentedFingerprint := parse(documented); documentedFingerprint != originalFingerprint {
		t.Fatalf("doc-only edit changed function fingerprint:\noriginal: %s\ndocumented: %s", originalFingerprint, documentedFingerprint)
	}
	if bodyChangedFingerprint := parse(changedBody); bodyChangedFingerprint == originalFingerprint {
		t.Fatalf("executable body edit did not change function fingerprint: %s", originalFingerprint)
	}
}

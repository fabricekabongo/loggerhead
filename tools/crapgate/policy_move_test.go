package main

import "testing"

func TestMovedFunctionUsesAliasForUnchangedBodyAndCoverageDelta(t *testing.T) {
	oldID := testID("pkg/old.go", "", "Stable")
	newID := testID("pkg/new.go", "", "Stable")
	baseBody := "func Stable() { return 1 }"
	baseFunctions := map[functionID]string{oldID: baseBody}
	headFunctions := map[functionID]string{newID: baseBody}
	aliases := map[functionID]functionID{newID: oldID}
	base := map[functionID]entry{oldID: testEntry(oldID, "Stable", 15)}
	head := map[functionID]entry{newID: testEntry(newID, "Stable", 16)}

	changed, err := evaluateWithRenamedBaseIDs(base, head, baseFunctions, headFunctions, aliases)
	if err != nil || len(changed) != 0 {
		t.Fatalf("unchanged moved function source results = %#v, err=%v", changed, err)
	}
	coverage, err := evaluateUnaffectedCRAPChanges(base, head, baseFunctions, headFunctions, aliases)
	if err != nil || len(coverage) != 1 || coverage[0].Allowed || coverage[0].Base != 15 || coverage[0].Head != 16 {
		t.Fatalf("moved function coverage result = %#v, err=%v", coverage, err)
	}
}

func TestMovedFunctionUsesAliasForChangedSourceMetrics(t *testing.T) {
	oldID := testID("pkg/old.go", "", "Stable")
	newID := testID("pkg/new.go", "", "Stable")
	base := map[functionID]entry{oldID: testEntryCC(oldID, "Stable", 15, 11)}
	head := map[functionID]entry{newID: testEntryCC(newID, "Stable", 14, 10)}
	baseFunctions := map[functionID]string{oldID: "func Stable() { return 1 }"}
	headFunctions := map[functionID]string{newID: "func Stable() { return 2 }"}
	aliases := map[functionID]functionID{newID: oldID}

	results, err := evaluateWithRenamedBaseIDs(base, head, baseFunctions, headFunctions, aliases)
	if err != nil || len(results) != 1 {
		t.Fatalf("changed moved function results = %#v, err=%v", results, err)
	}
	result := results[0]
	if result.Kind != "changed" || result.Base != 15 || result.Head != 14 || !result.Allowed {
		t.Fatalf("changed moved function result = %#v", result)
	}
}

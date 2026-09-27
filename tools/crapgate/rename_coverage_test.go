package main

import "testing"

func TestUnaffectedRenamedFunctionUsesBaseReportForCoverageDelta(t *testing.T) {
	oldID := testID("pkg/old.go", "", "Stable")
	newID := testID("pkg/new.go", "", "Stable")
	base := map[functionID]entry{oldID: testEntry(oldID, "Stable", 15)}
	head := map[functionID]entry{newID: testEntry(newID, "Stable", 16)}
	renamedBaseIDs := map[functionID]functionID{newID: oldID}
	baseFunctions := map[functionID]string{newID: "func Stable() { return 1 }"}
	headFunctions := map[functionID]string{newID: "func Stable() { return 1 }"}

	results, err := evaluateUnaffectedCRAPChanges(base, head, baseFunctions, headFunctions, renamedBaseIDs)
	if err != nil {
		t.Fatalf("evaluate renamed test-induced CRAP delta: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("renamed function CRAP results = %#v, want one changed result", results)
	}
	result := results[0]
	if result.ID != newID || result.Base != 15 || result.Head != 16 || result.Allowed {
		t.Fatalf("renamed function CRAP result = %#v, want old-path base 15, head 16, denied", result)
	}
}

package crapfixture

import "testing"

func TestScorePartial(t *testing.T) {
	Score(true, true, false, false)
}

func TestScoreFull(t *testing.T) {
	Score(true, true, true, true)
}

func TestCounterAdd(t *testing.T) {
	if got := (Counter{}).Add(1); got != 2 {
		t.Fatalf("Add(1) = %d, want 2", got)
	}
}

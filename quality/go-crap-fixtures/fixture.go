package crapfixture

// Score has cyclomatic complexity 5: four independent decisions.
func Score(a, b, c, d bool) {
	n := 0
	n++
	n++
	n++
	if a {
		n++
		touch()
	}
	if b {
		n++
		touch()
	}
	if c {
		n++
		touch()
	}
	if d {
		n++
	}
}

func touch() {}

type Counter struct{}

// Add exercises receiver-qualified function naming in reports.
func (Counter) Add(x int) int {
	if x > 0 {
		return x + 1
	}
	return x
}

// Uncovered has no test by design.
func Uncovered(x int) int {
	if x > 0 {
		return x * 2
	}
	return x
}

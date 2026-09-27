package main

import "testing"

func TestParseFunctionsSkipsDocumentationPackageBeforeBodyParsing(t *testing.T) {
	for _, source := range []string{
		"package documentation\nfunc Example() {}\n",
		"package documentation\nfunc Malformed( {\n",
	} {
		functions, err := parseFunctions("pkg/documentation.go", "", []byte(source))
		if err != nil {
			t.Fatalf("parse documentation package: %v", err)
		}
		if len(functions) != 0 {
			t.Fatalf("documentation package entered function inventory: %#v", functions)
		}
	}

	if _, err := parseFunctions("pkg/production.go", "", []byte("package production\nfunc Malformed( {\n")); err == nil {
		t.Fatal("malformed production package was ignored")
	}
	if _, err := parseFunctions("pkg/production.go", "", []byte("package\n")); err == nil {
		t.Fatal("malformed package clause was ignored")
	}
}

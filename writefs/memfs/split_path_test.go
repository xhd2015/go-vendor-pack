package memfs

import "testing"

// go test -run TestSplitPaths -v ./writefs/memfs
func TestSplitPaths(t *testing.T) {
	mustSplitPath := func(s string) []string {
		p, err := splitPath(s)
		if err != nil {
			panic(err)
		}
		return p
	}
	expectSplitPathErr(t, "", "empty pathName")
	expectList(t, `splitPath("/")`, mustSplitPath("/"), []string{})
	expectList(t, `splitPath("./")`, mustSplitPath("./"), []string{})
	expectList(t, `splitPath("/a/b/c")`, mustSplitPath("/a/b/c"), []string{"a", "b", "c"})
	expectList(t, `splitPath("/a/b//c")`, mustSplitPath("/a/b//c"), []string{"a", "b", "c"})
	expectList(t, `splitPath("a/b/c")`, mustSplitPath("a/b/c"), []string{"a", "b", "c"})
	expectList(t, `splitPath("./a/b/c")`, mustSplitPath("./a/b/c"), []string{"a", "b", "c"})
	// expectList(t, `splitPath("../a/b/c")`, mustSplitPath("../a/b/c"), []string{"b", "c"})
	// expectList(t, `splitPath("../../a/b/c")`, mustSplitPath("../a/b/c"), []string{"b", "c"})
	// expectList(t, `splitPath("../a/b/c")`, mustSplitPath("../a/b/c"), []string{"b", "c"})
	expectSplitPathErr(t, "../a/b/../c/..", "invalid relative path: ../a/b/../c/..")
	expectSplitPathErr(t, "..", "invalid relative path: ..")
	expectSplitPathErr(t, "../..", "invalid relative path: ../..")
}

func expectList(t *testing.T, name string, actual []string, expect []string) {
	if len(actual) != len(expect) {
		t.Fatalf("expect %s to be %v, actual: %v", name, expect, actual)
	}
	for i := 0; i < len(actual); i++ {
		if actual[i] != expect[i] {
			t.Fatalf("expect %s to be %v, actual: %v (differ at %v)", name, expect, actual, i)
		}
	}
}

func expectSplitPathErr(t *testing.T, path string, errMsg string) {
	paths, err := splitPath(path)
	if err == nil {
		t.Fatalf("expect splitPath(%q) error, actual returns: %v", path, paths)
	}
	actErr := err.Error()
	if actErr != errMsg {
		t.Fatalf("expect splitPath(%q) error: %s, actual error: %s", path, errMsg, actErr)
	}
}

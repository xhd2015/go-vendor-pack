package go_info

import "testing"

// go test -run TestGetGoVersion -v ./util/go_info
func TestGetGoVersion(t *testing.T) {
	version, err := GetGoVersion()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("version: %+v", version)
}

// go test -run TestCloneGo -v ./util/go_info
func TestCloneGo(t *testing.T) {
	goRoot, err := cloneGo(&GoVersion{
		Major: 1,
		Minor: 18,
		Patch: 1,
		OS:    "linux",
		Arch:  "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("clone go root: %+v", goRoot)
}

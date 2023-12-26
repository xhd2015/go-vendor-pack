package pack

import (
	"testing"
)

// go test -run TestPack -v ./pack
func TestPack(t *testing.T) {
	err := PackAsBase64ToCode("./testdata/source", "pack", "testPack", "packdata_test.go", &Options{})
	if err != nil {
		t.Fatal(err)
	}
}

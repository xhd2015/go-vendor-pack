package pack

import (
	"testing"
)

func TestPack(t *testing.T) {
	err := PackAsBase64ToCode("./testdata/source", "pack", "testPack", "packdata_test.go")
	if err != nil {
		t.Fatal(err)
	}
}

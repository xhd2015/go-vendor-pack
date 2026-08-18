package pack

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"testing"
)

// go test -run TestPack -v ./pack
func TestPack(t *testing.T) {
	err := PackAsBase64ToCode("./testdata/source", "pack", "testPack", "packdata_test.go", &Options{})
	if err != nil {
		t.Fatal(err)
	}
}

// PackAsBase64 must Close the base64 encoder. Without Close, the last 1–2
// gzip bytes are dropped whenever the compressed length is not a multiple of 3,
// and gzip.NewReader fails with unexpected EOF.
func TestPackAsBase64GzipRoundTrip(t *testing.T) {
	encoded, err := PackAsBase64("./testdata/source", &Options{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	if _, err := io.Copy(io.Discard, zr); err != nil {
		t.Fatalf("gzip read (encoder Close missing?): %v", err)
	}
	if err := zr.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
}


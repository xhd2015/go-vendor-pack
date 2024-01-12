package writefs

import (
	"io"
	"io/fs"
	"os"
	"time"
)

var _ FS = NoopFS{}
var _ FSWithTime = NoopFS{}

type NoopFS struct {
}

func (NoopFS) Stat(name string) (fs.FileInfo, error) {
	return nil, os.ErrNotExist
}
func (NoopFS) RemoveAll(name string) error {
	return nil
}

func (NoopFS) MkdirAll(name string, perm os.FileMode) error {
	return nil
}

func (NoopFS) OpenFileWrite(name string) (io.WriteCloser, error) {
	return discard{}, nil
}

// OpenFileRead implements FSWithTime.
func (NoopFS) OpenFileRead(name string) (io.ReadCloser, error) {
	return discard{}, nil
}

func (NoopFS) OpenFileAppend(name string) (io.WriteCloser, error) {
	return discard{}, nil
}

type discard struct{}

func (discard) Write(s []byte) (int, error) {
	return len(s), nil
}

func (discard) Read(p []byte) (n int, err error) {
	return 0, nil
}
func (discard) Close() error {
	return nil
}

// RemoveFile implements FS.
func (NoopFS) RemoveFile(name string) error {
	return nil
}

// ReadDir implements TreeFS.
func (NoopFS) ReadDir(name string) ([]fs.FileInfo, error) {
	return nil, nil
}

// Chtimes implements FSWithTime.
func (NoopFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return nil
}

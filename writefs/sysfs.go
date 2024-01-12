package writefs

import (
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"time"
)

type SysFS struct {
}

var _ FS = SysFS{}
var _ FSWithTime = SysFS{}

func (SysFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}
func (SysFS) RemoveAll(name string) error {
	return os.RemoveAll(name)
}

func (SysFS) RemoveFile(name string) error {
	return os.Remove(name)
}

func (SysFS) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(name, perm)
}
func (SysFS) OpenFileRead(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

func (SysFS) OpenFileWrite(name string) (io.WriteCloser, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
}

func (SysFS) OpenFileAppend(name string) (io.WriteCloser, error) {
	return os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0755)
}

func (SysFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return os.Chtimes(name, atime, mtime)
}

// ReadDir implements TreeFS.
func (SysFS) ReadDir(name string) ([]fs.FileInfo, error) {
	return ioutil.ReadDir(name)
}

package writefs

import (
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"time"
)

type FS interface {
	Stat(name string) (fs.FileInfo, error)

	RemoveFile(name string) error

	MkdirAll(name string, perm os.FileMode) error

	OpenFileRead(name string) (io.ReadCloser, error)
	OpenFileWrite(name string) (io.WriteCloser, error)
	OpenFileAppend(name string) (io.WriteCloser, error)

	RemoveAll(name string) error
	ReadDir(name string) ([]fs.FileInfo, error)
}

type FSWithTime interface {
	FS
	Chtimes(name string, atime time.Time, mtime time.Time) error
}

func IsNotExist(err error) bool {
	// return os.IsNotExist(err) : not work for wrapping
	return errors.Is(err, os.ErrNotExist)
}

func ReadFile(fs FS, name string) ([]byte, error) {
	read, err := fs.OpenFileRead(name)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(read)
	read.Close()
	if err != nil {
		return nil, err
	}
	return content, nil
}
func WriteFile(fs FS, name string, data []byte) error {
	w, err := fs.OpenFileWrite(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	closeErr := w.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

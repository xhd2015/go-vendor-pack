package packfs

import (
	"io/fs"
)

type ErrKind int

const (
	ErrKind_NotExists ErrKind = 1
)

type Error struct {
	Kind ErrKind
	Err  error
}

func NewError(kind ErrKind, err error) *Error {
	return &Error{Kind: kind, Err: err}
}
func (c *Error) Error() string {
	return c.Err.Error()
}

// FS can be fs embed.FS or tarball
type FS interface {
	ReadFile(file string) ([]byte, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

func IsNotExists(err error) bool {
	if fsErr, ok := err.(*Error); ok {
		return fsErr.Kind == ErrKind_NotExists
	}
	return false
}

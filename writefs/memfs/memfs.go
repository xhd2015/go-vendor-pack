package memfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/xhd2015/go-vendor-pack/writefs"
)

// MemFS is a thread safe in-memory filesystem
// NOTE: threadsafe is guranteed for dir operations, not file
// read and write

type MemFS struct {
	root *dirEntry
}

var _ writefs.FS = (*MemFS)(nil)

func New() *MemFS {
	return &MemFS{
		root: &dirEntry{
			entryType:   entryType_dir,
			childrenMap: make(map[string]*dirEntry),
			perm:        0755,
		},
	}
}

// Stat implements FS.
func (c *MemFS) Stat(name string) (fs.FileInfo, error) {
	entry, err := navDir(name, c.root, false)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *MemFS) MkdirAll(name string, perm fs.FileMode) error {
	entry, err := navDir(name, c.root, true)
	if err != nil {
		return err
	}
	entry.perm = perm
	return nil
}

func (c *MemFS) OpenFileWrite(name string) (io.WriteCloser, error) {
	return c.openFileWrite(name, true)
}

func (c *MemFS) OpenFileAppend(name string) (io.WriteCloser, error) {
	return c.openFileWrite(name, false)
}
func (c *MemFS) openFileWrite(name string, reset bool) (io.WriteCloser, error) {
	entry, baseName, err := navParent(name, c.root)
	if err != nil {
		return nil, err
	}
	entry.mutex.Lock()
	f := entry.childrenMap[baseName]
	if f == nil {
		f = &dirEntry{
			name:      baseName,
			parent:    entry,
			entryType: entryType_file,
			perm:      0755,
		}
		entry.children = append(entry.children, f)
		entry.childrenMap[baseName] = f
	}
	if reset {
		f.buf.Reset()
	}
	entry.mutex.Unlock()

	return &f.buf, nil
}

// OpenFileRead implements writefs.FS.
func (c *MemFS) OpenFileRead(name string) (io.ReadCloser, error) {
	entry, err := navDir(name, c.root, false)
	if err != nil {
		return nil, err
	}
	// read buf will advance buf.off, which
	// will change state, so we make
	// a state copy here
	buf := entry.buf

	return &buf, nil
}

func (c *MemFS) TraversePath(fn func(path string, e MemFileInfo) bool) {
	c.Traverse("", func(prefixCtx interface{}, e MemFileInfo) (interface{}, bool) {
		subPath := prefixCtx.(string) + "/" + e.Name()
		ok := fn(subPath, e)
		return subPath, ok
	})
}

func (c *MemFS) Traverse(initCtx interface{}, fn func(ctx interface{}, e MemFileInfo) (subCtx interface{}, ok bool)) {
	var traverse func(ctx interface{}, e *dirEntry) bool
	traverse = func(ctx interface{}, e *dirEntry) bool {
		var subCtx interface{} = ctx
		if e != c.root {
			var ok bool
			subCtx, ok = fn(ctx, e)
			if !ok {
				return false
			}
		}
		e.mutex.Lock()
		copyChildren := make([]*dirEntry, len(e.children))
		copy(copyChildren, e.children)
		e.mutex.Unlock()

		for _, child := range copyChildren {
			if !traverse(subCtx, child) {
				return false
			}
		}
		return true
	}
	traverse(initCtx, c.root)
}

// ReadDir implements FS.
func (c *MemFS) ReadDir(name string) ([]fs.FileInfo, error) {
	entry, err := navDir(name, c.root, false)
	if err != nil {
		return nil, err
	}
	entry.mutex.Lock()
	infos := make([]fs.FileInfo, 0, len(entry.children))
	for _, child := range entry.children {
		infos = append(infos, child)
	}
	entry.mutex.Unlock()

	return infos, nil
}

func (c *MemFS) RemoveAll(name string) error {
	return c.remove(name, removeMode_all)
}

func (c *MemFS) RemoveFile(name string) error {
	return c.remove(name, removeMode_file)
}

type removeMode int

const (
	removeMode_file removeMode = 0
	removeMode_all  removeMode = 1
)

func (c *MemFS) remove(name string, removeMode removeMode) error {
	entry, baseName, err := navParent(name, c.root)
	if err != nil {
		if writefs.IsNotExist(err) && removeMode == removeMode_all {
			return nil
		}
		return err
	}
	entry.mutex.Lock()
	defer entry.mutex.Unlock()
	f := entry.childrenMap[baseName]
	if f != nil {
		if removeMode == removeMode_file {
			if f.entryType == entryType_dir && len(f.children) > 0 {
				return fmt.Errorf("rm non-empty dir: %s", name)
			}
		}
		delete(entry.childrenMap, baseName)
		n := len(entry.children)
		for i := 0; i < n; i++ {
			if entry.children[i] == f {
				for j := i + 1; j < n; j++ {
					entry.children[j-1] = entry.children[j]
				}
				entry.children = entry.children[:n-1]
				break
			}
		}
	} else if removeMode == removeMode_file {
		return fmt.Errorf("rm %s: %w", name, os.ErrNotExist)
	}

	return nil
}

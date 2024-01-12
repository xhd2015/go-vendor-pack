package memfs

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var _ fs.FileInfo = (*dirEntry)(nil)
var _ MemFileInfo = (*dirEntry)(nil)

type MemFileInfo interface {
	fs.FileInfo

	Parent() MemFileInfo
	Buffer() *bytes.Buffer
	GetData() interface{}
	SetData(data interface{})
}

type entryType int

const (
	entryType_dir  entryType = 0
	entryType_file entryType = 1
)

type dirEntry struct {
	mutex     sync.Mutex
	name      string
	entryType entryType

	parent *dirEntry

	children    []*dirEntry
	childrenMap map[string]*dirEntry

	perm os.FileMode

	buf  buf
	data interface{} // associated data
}

type buf struct {
	bytes.Buffer
}

func (c *buf) Close() error {
	return nil
}

func (c *dirEntry) IsDir() bool {
	return c.entryType == entryType_dir
}

func (c *dirEntry) ModTime() time.Time {
	return time.Time{}
}

func (c *dirEntry) Mode() fs.FileMode {
	return c.perm
}

func (c *dirEntry) Name() string {
	return c.name
}

func (c *dirEntry) Size() int64 {
	return int64(c.buf.Len())
}

// Sys implements fs.FileInfo.
func (c *dirEntry) Sys() interface{} {
	return nil
}
func (c *dirEntry) Parent() MemFileInfo {
	if c.parent == nil {
		return nil
	}
	return c.parent
}
func (c *dirEntry) GetData() interface{} {
	return c.data
}
func (c *dirEntry) SetData(data interface{}) {
	c.data = data
}

func (c *dirEntry) Buffer() *bytes.Buffer {
	return &c.buf.Buffer
}

func navParent(path string, root *dirEntry) (*dirEntry, string, error) {
	return navPath(path, true, root, false)
}
func navDir(path string, root *dirEntry, create bool) (*dirEntry, error) {
	entry, _, err := navPath(path, false, root, create)
	return entry, err
}
func navPath(path string, parent bool, root *dirEntry, create bool) (entry *dirEntry, baseName string, err error) {
	names, err := splitPath(path)
	if err != nil {
		return nil, "", err
	}
	// assert root!=nil
	p := root
	n := len(names)
	if parent {
		if n == 0 {
			return nil, "", fmt.Errorf("no parent %s:%w", path, os.ErrNotExist)
		}
		baseName = names[n-1]
		n--
	}
	for i := 0; i < n; i++ {
		name := names[i]
		p.mutex.Lock()
		next := p.childrenMap[name]
		p.mutex.Unlock()
		if next == nil {
			if !create {
				return nil, "", fmt.Errorf("%s: %w", filepath.Join(names[:i+1]...), os.ErrNotExist)
			}
			if p.entryType != entryType_dir {
				// TODO return
				return nil, "", fmt.Errorf("not dir: %s", filepath.Join(names[:i+1]...))
			}
			// create all
			for j := i; j < n; j++ {
				dirName := names[j]
				next = &dirEntry{
					name:        dirName,
					parent:      p,
					entryType:   entryType_dir,
					childrenMap: make(map[string]*dirEntry),
				}
				p.mutex.Lock()
				p.children = append(p.children, next)
				p.childrenMap[dirName] = next
				p.mutex.Unlock()
				p = next
			}
			return p, baseName, nil
		}
		p = next
	}
	return p, baseName, nil
}

var pathSplitor = regexp.MustCompile(`[/\\]`)

func splitPath(path string) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("empty pathName")
	}
	// supports:
	//  /a/b/c ->  root=true,   a,b,c
	//  a/b/c ->   root=false,  a,b,c
	// ./a/b/c ->  root=false,  a,b,c
	// ../a/b/c -> root=false,  ..,a,b,c
	//

	splitList := pathSplitor.Split(path, -1)

	// compact the array
	j := 0
	n := len(splitList)

	for i := 0; i < n; i++ {
		e := splitList[i]
		if e == "" || e == "." {
			continue
		}
		if e == ".." {
			return nil, fmt.Errorf("invalid relative path: %s", path)
		}
		splitList[j] = splitList[i]
		j++
	}
	return splitList[:j], nil
}

package map_fs

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/go-vendor-pack/packfs"
)

type MapFS struct {
	dirs  map[string][]fs.DirEntry
	files map[string][]byte
}

func New() *MapFS {
	return &MapFS{
		dirs:  make(map[string][]fs.DirEntry),
		files: make(map[string][]byte),
	}
}

// TODO: this is not memory efficient
func NewFromDir(dir string, prefix string) (*MapFS, error) {
	mapFS := New()
	dir = strings.TrimPrefix(dir, "./")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		// return on any error
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(strings.TrimPrefix(path, dir), string(filepath.Separator))
		if prefix != "" {
			name = filepath.Join(prefix, name)
		}
		if !d.IsDir() {
			// open files for taring
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			content, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			mapFS.files[name] = content
		}

		dirName := filepath.Dir(name)
		mapFS.dirs[dirName] = append(mapFS.dirs[dirName], d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mapFS, nil
}

func (c *MapFS) SetDir(dir string, files []fs.DirEntry) {
	c.dirs[dir] = files
}

func (c *MapFS) SetFile(file string, content []byte) {
	c.files[file] = content
}

func (c *MapFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, ok := c.dirs[name]
	if !ok {
		return nil, packfs.NewError(packfs.ErrKind_NotExists, fmt.Errorf("dir not found: %s", name))
	}
	return entries, nil
}
func (c *MapFS) ReadFile(file string) ([]byte, error) {
	content, ok := c.files[file]
	if !ok {
		return nil, packfs.NewError(packfs.ErrKind_NotExists, fmt.Errorf("file not found: %s", file))
	}
	return content, nil
}

type DirEntry struct {
	fs.DirEntry
	DirFlag bool
	DirName string
}

func (c *DirEntry) Name() string {
	return c.DirName
}
func (c *DirEntry) IsDir() bool {
	return c.DirFlag
}

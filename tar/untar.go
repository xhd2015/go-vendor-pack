package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/go-vendor-pack/packfs"
)

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) error {
	return ForEachFileInTar(r, func(header *tar.Header, tarReader io.Reader) (error, bool) {
		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)
		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err, false
				}
			}
		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err, false
			}

			// copy over contents
			if _, err := io.Copy(f, tarReader); err != nil {
				return err, false
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
		return nil, true
	})
}

type tarFS struct {
	mapping map[string]*info
}

type info struct {
	header   *tar.Header
	content  []byte
	self     *dirEntry
	children []fs.DirEntry
}

type dirEntry struct {
	fs.DirEntry
	name  string
	isDir bool
}

func (c *dirEntry) Name() string {
	return c.name
}
func (c *dirEntry) IsDir() bool {
	return c.isDir
}

type fileInfo struct {
	fs.FileInfo
}

var _ packfs.FS = (*tarFS)(nil)

func NewTarFS(r io.Reader) (packfs.FS, error) {
	mapping := make(map[string]*info)
	err := ForEachFileInTar(r, func(header *tar.Header, r io.Reader) (error, bool) {
		content, err := ioutil.ReadAll(r)
		if err != nil {
			return err, false
		}
		name := normalize(header.Name)
		mapping[name] = &info{
			header: header,
			self: &dirEntry{
				name:  basename(name),
				isDir: header.Typeflag == tar.TypeDir,
			},
			content: content,
		}
		return nil, true
	})
	if err != nil {
		return nil, err
	}
	err = fillTree(mapping)
	if err != nil {
		return nil, err
	}
	return &tarFS{
		mapping: mapping,
	}, nil
}

func fillTree(mapping map[string]*info) error {
	for name, inf := range mapping {
		parent := dirname(name)
		if parent == "" || parent == "." || parent == "/" {
			continue
		}
		parentInf := mapping[parent]
		if parentInf == nil {
			return fmt.Errorf("building tree: %s not found", parent)
		}
		parentInf.children = append(parentInf.children, inf.self)
	}
	return nil
}

func basename(s string) string {
	if s == "" || s == "/" {
		return s
	}
	idx := strings.LastIndex(s, "/")
	if idx < 0 {
		return s
	}
	return s[idx+1:]
}

func dirname(s string) string {
	if s == "" || s == "/" {
		return ""
	}
	idx := strings.LastIndex(s, "/")
	if idx < 0 {
		return ""
	}
	return s[:idx]
}

func normalize(name string) string {
	name = strings.TrimPrefix(name, "./")
	if name != "/" {
		name = strings.TrimSuffix(name, "/")
	}
	return name
}

func dirPrefixWith(dir string, prefix string) bool {
	if len(dir) <= len(prefix) {
		return false
	}
	if !strings.HasPrefix(dir, prefix) {
		return false
	}
	return dir[len(prefix)] == '/'
}

// ReadDir implements helper.FS.
func (t *tarFS) ReadDir(name string) ([]fs.DirEntry, error) {
	inf, ok := t.mapping[name]
	if !ok {
		return nil, packfs.NewError(packfs.ErrKind_NotExists, fmt.Errorf("no such directory: %v", name))
	}
	if inf.header.Typeflag != tar.TypeDir {
		return nil, fmt.Errorf("type error, expecting directory, actual file: %v", name)
	}
	return inf.children, nil
}

// ReadFile implements helper.FS.
func (t *tarFS) ReadFile(file string) ([]byte, error) {
	inf, ok := t.mapping[file]
	if !ok {
		return nil, packfs.NewError(packfs.ErrKind_NotExists, fmt.Errorf("no such file: %v", file))
	}
	if inf.self.isDir {
		return nil, fmt.Errorf("not a file: %v", file)
	}
	return inf.content, nil
}

func ForEachFileInTar(r io.Reader, fn func(header *tar.Header, r io.Reader) (error, bool)) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}
		err, ok := fn(header, tr)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
}

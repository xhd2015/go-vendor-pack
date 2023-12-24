package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TarOptions struct {
	ShouldInclude func(relPath string, dir bool) bool
	OnAdd         func(relPath string, dir bool)
	WritePrefix   string
	ClearModTime  bool
}

// Tar takes a source and variable writers and walks 'source' writing each file
// found to the tar writer; the purpose for accepting multiple writers is to allow
// for multiple outputs (for example a file, or md5 hash)
func Tar(src string, writer io.Writer, opts *TarOptions) error {
	tw, close := WrapTarWriter(writer)
	defer close()
	return TarAppend(src, tw, opts)
}
func WrapTarWriter(writer io.Writer) (tw *tar.Writer, close func() error) {
	mw := writer

	gzw := gzip.NewWriter(mw)

	tw = tar.NewWriter(gzw)
	close = func() error {
		twErr := tw.Close()
		gzErr := gzw.Close()
		if twErr != nil {
			return twErr
		}
		if gzErr != nil {
			return gzErr
		}
		return nil
	}
	return
}

func TarAppend(src string, tw *tar.Writer, opts *TarOptions) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files - %v", err.Error())
	}

	src = strings.TrimPrefix(src, "./")

	// walk path, path start with src
	return filepath.WalkDir(src, func(path string, info fs.DirEntry, err error) error {
		// return on any error
		if err != nil {
			return err
		}

		finfo, err := info.Info()
		if err != nil {
			return err
		}
		isDir := info.Type().IsDir()

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !isDir && !info.Type().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(finfo, info.Name())
		if err != nil {
			return err
		}
		// set zero time(to avoid tar content change for every generate)
		header.ModTime = time.Time{}

		// update the name to correctly reflect the desired destination when untaring
		name := strings.TrimPrefix(strings.TrimPrefix(path, src), string(filepath.Separator))
		if opts != nil {
			name = filepath.Join(opts.WritePrefix, name)
		}
		if name == "" {
			// the root
			return nil
		}
		header.Name = name

		if false {
			log.Printf("tar check: %s isDir=%v", name, isDir)
		}
		if opts != nil && opts.ShouldInclude != nil {
			if !opts.ShouldInclude(name, isDir) {
				if isDir {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if opts != nil && opts.OnAdd != nil {
			opts.OnAdd(name, isDir)
		}
		if false {
			log.Printf("tar add: %s isDir=%v", name, isDir)
		}

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if isDir {
			return nil
		}

		// open files for taring
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}

func TarAdd(tw *tar.Writer, header *tar.Header, content io.Reader) error {
	// write the header
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if header.Typeflag == tar.TypeDir || content == nil {
		return nil
	}
	_, err := io.Copy(tw, content)
	return err
}

// no modTime included
func TarAddDir(tw *tar.Writer, name string, mode fs.FileMode) error {
	return TarAdd(tw, &tar.Header{
		Typeflag: tar.TypeDir,
		Name:     name,
		Mode:     int64(mode),
	}, nil)
}

package helper

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/xhd2015/go-vendor-pack/packfs"
)

// must ensure HasVendor
func AddVendor(dir string, module string, fs packfs.FS, overrideAll bool, overrideSubPath map[string]bool) (added bool, err error) {
	vendorName := filepath.Join("vendor", module)
	err = copyDirOverrideFiles(fs, vendorName, filepath.Join(dir, vendorName), func(subPath string) bool {
		return overrideAll || overrideSubPath[subPath]
	})
	added = true
	return
}

// copyDirOverrideFiles will override files, but do not delete dirs
// equivalent: cp name/* dir/*
// e.g. copy anything under name to dir
// this copy is aware of go's module inclusion logic, where files form a package, not dirs.
// it treats all files as a unit, and either replace them all or just change nothing.
func copyDirOverrideFiles(fs packfs.FS, name string, dir string, shouldOverrideFiles func(subPath string) bool) error {
	var copyDir func(name string, dir string, relPath string) error
	copyDir = func(name string, dir string, relPath string) error {
		dirExists := true
		_, statErr := os.Stat(dir)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				dirExists = false
			} else {
				return statErr
			}
		}

		srcEntries, err := fs.ReadDir(name)
		if err != nil {
			return err
		}
		var srcFiles []string
		var srcDirs []string
		// copy all source files
		for _, entry := range srcEntries {
			subName := entry.Name()
			if entry.IsDir() {
				srcDirs = append(srcDirs, subName)
			} else {
				srcFiles = append(srcFiles, subName)
			}
		}

		if len(srcFiles) > 0 {
			var override bool
			if !dirExists {
				log.Printf("package added: %v", name)
			} else {
				override = shouldOverrideFiles != nil && shouldOverrideFiles(relPath)
				if override {
					log.Printf("package override: %v", name)
				} else {
					log.Printf("package reuse: %v", name)
				}
			}

			// override files when dir does not exist or should be overridden
			if !dirExists || override {
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					return err
				}

				dstEntries, err := ioutil.ReadDir(dir)
				if err != nil {
					return err
				}
				// remove all dst files, except dirs
				for _, dstEntry := range dstEntries {
					if dstEntry.IsDir() {
						continue
					}
					err := os.Remove(path.Join(dir, dstEntry.Name()))
					if err != nil {
						return fmt.Errorf("remove file in original directory: %w", err)
					}
				}
				// copy all source files
				for _, srcFileName := range srcFiles {
					subFile := path.Join(dir, srcFileName)
					targetName := filepath.Join(name, srcFileName)
					err := CopyFile(fs, targetName, subFile)
					if err != nil {
						return err
					}
				}
			}
		}
		// check all sub directorys
		for _, srcDirName := range srcDirs {
			subFile := path.Join(dir, srcDirName)
			targetName := filepath.Join(name, srcDirName)
			err := copyDir(targetName, subFile, relPath+"/"+srcDirName)
			if err != nil {
				return err
			}
		}

		return nil
	}
	return copyDir(name, dir, "")
}

// CopyFile
// NOTE: must ensure dst's parent exists
func CopyFile(fs packfs.FS, name string, dst string) error {
	content, err := fs.ReadFile(name)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dst, content, 0755)
	if err != nil {
		return err
	}
	return nil
}

func HasVendor(dir string) bool {
	stat, err := os.Stat(filepath.Join(dir, "vendor"))
	return err == nil && stat.IsDir()
}

func exists(f string) (bool, error) {
	_, err := os.Stat(f)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

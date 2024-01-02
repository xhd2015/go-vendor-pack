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

func OverrideFiles(fs packfs.FS, srcDir string, dstDir string) error {
	_, err := overrideFiles(fs, srcDir, dstDir, true)
	return err
}
func CopyFiles(fs packfs.FS, name string, dir string, shouldOverrideFiles func(subPath string) bool) error {
	return copyDirOverrideFiles(fs, name, dir, shouldOverrideFiles)
}

// copyDirOverrideFiles will override files, but do not delete dirs
// equivalent: cp name/* dir/*
// e.g. copy anything under name to dir
// this copy is aware of go's module inclusion logic, where files form a package, not dirs.
// it treats all files as a unit, and either replace them all or just change nothing.
func copyDirOverrideFiles(fs packfs.FS, name string, dir string, shouldOverrideFiles func(subPath string) bool) error {
	var copyDir func(name string, dir string, relPath string) error
	copyDir = func(name string, dir string, relPath string) error {
		srcDirs, err := overrideFiles(fs, name, dir, shouldOverrideFiles != nil && shouldOverrideFiles(relPath))
		if err != nil {
			return err
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

func overrideFiles(fs packfs.FS, srcFsPath string, dstDir string, override bool) (srcDirs []string, err error) {
	var srcFiles []string
	srcFiles, srcDirs, err = readEntries(fs, srcFsPath)
	if err != nil {
		return
	}

	if len(srcFiles) == 0 {
		return
	}
	var dirExists bool
	dirExists, err = checkDirForRemoving(dstDir, override)
	if err != nil {
		err = fmt.Errorf("remove file in original directory: %w", err)
		return
	}

	if !dirExists {
		log.Printf("package added: %v", srcFsPath)
		err = os.MkdirAll(dstDir, 0755)
		if err != nil {
			return
		}
	} else {
		if override {
			log.Printf("package override: %v", srcFsPath)
		} else {
			log.Printf("package reuse: %v", srcFsPath)
			return
		}
	}
	// override files when dir does not exist or should be overridden
	// copy all source files
	for _, srcFileName := range srcFiles {
		subFile := path.Join(dstDir, srcFileName)
		targetName := filepath.Join(srcFsPath, srcFileName)
		err = CopyFile(fs, targetName, subFile)
		if err != nil {
			return
		}
	}
	return
}

func checkDirForRemoving(dir string, override bool) (dirExists bool, err error) {
	if !override {
		_, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	dstEntries, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	// remove all dst files, except dirs
	for _, dstEntry := range dstEntries {
		if dstEntry.IsDir() {
			continue
		}
		err := os.Remove(path.Join(dir, dstEntry.Name()))
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func readEntries(fs packfs.FS, name string) (srcFiles []string, srcDirs []string, err error) {
	srcEntries, err := fs.ReadDir(name)
	if err != nil {
		return
	}
	// copy all source files
	for _, entry := range srcEntries {
		subName := entry.Name()
		if entry.IsDir() {
			srcDirs = append(srcDirs, subName)
		} else {
			srcFiles = append(srcFiles, subName)
		}
	}
	return
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

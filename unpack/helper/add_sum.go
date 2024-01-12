package helper

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/go-vendor-pack/go_cmd"
	"github.com/xhd2015/go-vendor-pack/writefs"
)

func AddVersionAndSum(dir string, mod string, version string, sum string, replace string) error {
	return AddVersionAndSumFS(writefs.SysFS{}, dir, mod, version, sum, replace)
}
func AddVersionAndSumFS(fs writefs.FS, dir string, mod string, version string, sum string, replace string) error {
	editGoMod := func(goModFile string) error {
		err := go_cmd.GoModRequire(goModFile, mod, version)
		if err != nil {
			return err
		}
		if replace != "" {
			err := go_cmd.GoModReplace(goModFile, mod, replace)
			if err != nil {
				return err
			}
		}
		return nil
	}
	goModFile := filepath.Join(dir, "go.mod")
	if _, ok := fs.(writefs.SysFS); ok {
		// go mod edit
		err := editGoMod(goModFile)
		if err != nil {
			return err
		}
	} else {
		reader, err := fs.OpenFileRead(goModFile)
		if err != nil {
			return err
		}
		defer reader.Close()
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		newContent, err := go_cmd.GoModEdit(string(content), editGoMod)
		if err != nil {
			return err
		}
		w, err := fs.OpenFileWrite(goModFile)
		if err != nil {
			return err
		}
		defer w.Close()
		_, err = w.Write([]byte(newContent))
		if err != nil {
			return err
		}
	}

	addGoSum := func() error {
		// append to go sum
		sumFile := filepath.Join(dir, "go.sum")

		writer, err := fs.OpenFileAppend(sumFile)
		if err != nil {
			return err
		}
		defer writer.Close()

		writer.Write([]byte(sum))
		if !strings.HasSuffix(sum, "\n") {
			writer.Write([]byte("\n"))
		}
		return nil
	}
	if sum != "" {
		err := addGoSum()
		if err != nil {
			return fmt.Errorf("updating go.sum: %w", err)
		}
	}

	// update modules.txt
	modulesFile := filepath.Join(dir, "vendor/modules.txt")
	modFileReader, err := fs.OpenFileRead(modulesFile)
	if err != nil {
		if os.IsNotExist(err) {
			// skip optional vendopr
			return nil
		}
		return err
	}
	defer modFileReader.Close()

	modulesContent, err := ioutil.ReadAll(modFileReader)
	if err != nil {
		return err
	}

	// in vendor/modules.txt
	//    after
	//       # mod version
	//    either
	//       ## explicit
	//       mod
	//    or a list of packages like:
	//       mod/pkgA
	//       mod/pkgB
	//   ....
	lines := strings.Split(string(modulesContent), "\n")
	n := len(lines)
	prefixModSpace := "# " + mod + " "
	// find previous explicit
	i := 0
	for ; i < n; i++ {
		line := lines[i]
		if strings.HasPrefix(line, prefixModSpace) {
			// found module declaration
			break
		}
	}
	// example:
	//   # githuh.com/example/support/tls v1.0.1
	//   ## explicit
	//   githuh.com/example/support/tls
	var needAppendExplicit bool
	if i >= n {
		// not found any declaration, add new explicit
		needAppendExplicit = true
	} else {
		// force replace version
		if i+1 >= n || strings.TrimSpace(lines[i+1]) != "## explicit" {
			// found detailed list, remove all packages until next module(starts with #)
			// replace with explicit
			for j := i + 1; j <= n; j++ {
				if j >= n || strings.HasPrefix(lines[j], "#") {
					// the range of [i,j-1] should be removed
					m := j - i
					for ; j < n; j++ {
						lines[j-m] = lines[j]
					}
					lines = lines[:n-m]
					break
				}
			}
			needAppendExplicit = true
		} else {
			lines[i] = prefixModSpace + version
		}
	}
	if needAppendExplicit {
		// format:
		//   # MODULE VERSION
		//   ## explicit
		//   # PKG
		lines = append(lines,
			prefixModSpace+version,
			"## explicit",
			mod,
		)
	}

	newModulesContent := strings.Join(lines, "\n")

	w, err := fs.OpenFileWrite(modulesFile)
	if err != nil {
		return fmt.Errorf("adding package error:%v %v", mod, err)
	}
	defer w.Close()
	w.Write([]byte(newModulesContent))
	if err != nil {
		return fmt.Errorf("adding package error:%v %v", mod, err)
	}
	if verboseLog {
		log.Printf("module added: %v %v", mod, version)
	}
	return nil
}

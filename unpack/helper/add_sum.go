package helper

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xhd2015/go-inspect/sh"
)

func AddVersionAndSum(dir string, mod string, version string, sum string) error {
	// go mod edit
	_, _, err := sh.RunBashWithOpts([]string{
		`go mod edit -require="$IMPORT_PATH@$IMPORT_VERSION"`,
	}, sh.RunBashOptions{

		FilterCmd: func(cmd *exec.Cmd) {
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "IMPORT_PATH="+mod)
			cmd.Env = append(cmd.Env, "IMPORT_VERSION="+version)
			cmd.Dir = dir
		},
	})
	if err != nil {
		return err
	}

	addGoSum := func() error {
		// append to go sum
		sumFile := filepath.Join(dir, "go.sum")
		file, err := os.OpenFile(sumFile, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		defer file.Close()

		file.WriteString(sum)
		if !strings.HasSuffix(sum, "\n") {
			file.WriteString("\n")
		}
		return nil
	}
	err = addGoSum()
	if err != nil {
		return fmt.Errorf("updating go.sum: %w", err)
	}

	// update modules.txt
	modulesFile := filepath.Join(dir, "vendor/modules.txt")
	modulesContent, err := ioutil.ReadFile(modulesFile)
	if err != nil {
		if os.IsNotExist(err) {
			// skip optional vendopr
			return nil
		}
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

	err = ioutil.WriteFile(modulesFile, []byte(newModulesContent), 0755)
	if err != nil {
		return fmt.Errorf("adding package error:%v %v", mod, err)
	}
	log.Printf("module added: %v %v", mod, version)
	return nil
}

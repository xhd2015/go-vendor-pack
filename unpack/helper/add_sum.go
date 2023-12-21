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

	lines := strings.Split(string(modulesContent), "\n")
	n := len(lines)
	prefix := "# " + mod + " "
	// find previous explicit
	i := 0
	for ; i < n; i++ {
		line := lines[i]
		if strings.HasPrefix(line, prefix) {
			if i+1 < n {
				nextLine := lines[i+1]
				if strings.TrimSpace(nextLine) == "## explicit" {
					// found i
					break
				}
			}
		}
	}
	// example:
	//   # githuh.com/example/support/tls v1.0.1
	//   ## explicit
	//   githuh.com/example/support/tls
	if i < n {
		lines[i] = prefix + version
	} else {
		// format:
		//   # MODULE VERSION
		//   ## explicit
		//   # PKG
		lines = append(lines,
			prefix+version,
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

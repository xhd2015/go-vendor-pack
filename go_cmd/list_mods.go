package go_cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/xhd2015/go-vendor-pack/go_cmd/model"
)

// NOTE: go list -deps -json will only reports package required by current
// go version, that is not sufficient for a general pureposed packer.
// However, it is found that `go list -m -mod=readonly -json all`
// would report modules that does not appear in vendor, this is
// the main problem why it is not useful.
func ListAllModules(dir string) ([]*model.ModulePublic, error) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer

	// NOTE: cannot use go list -m in vendor mode, so force -mod=readonly
	cmd := exec.Command("go", "list", "-m", "-mod=readonly", "-json", "all")
	cmd.Dir = dir
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("go list:%w %v", err, errBuf.String())
	}

	decoder := json.NewDecoder(&buf)

	var modules []*model.ModulePublic
	for decoder.More() {
		var mod model.ModulePublic
		err := decoder.Decode(&mod)
		if err != nil {
			return nil, fmt.Errorf("decoding pkg: %w", err)
		}
		modules = append(modules, &mod)
	}
	return modules, nil
}

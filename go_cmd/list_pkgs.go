package go_cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/xhd2015/go-vendor-pack/go_cmd/model"
)

// NOTE: go list -deps -json will only reports package required by current
// go version, that is not sufficient for a general pureposed packer
func ListPackages(dir string, args ...string) ([]*model.PackagePublic, error) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer

	// NOTE: by default test packages not included(included by go list -test)
	listArgs := append([]string{"list", "-deps", "-json"}, args...)
	cmd := exec.Command("go", listArgs...)
	cmd.Dir = dir
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("go list:%w %v", err, errBuf.String())
	}

	decoder := json.NewDecoder(&buf)

	var pkgs []*model.PackagePublic
	for decoder.More() {
		var pkg model.PackagePublic
		err := decoder.Decode(&pkg)
		if err != nil {
			return nil, fmt.Errorf("decoding pkg: %w", err)
		}
		pkgs = append(pkgs, &pkg)
	}
	return pkgs, nil
}

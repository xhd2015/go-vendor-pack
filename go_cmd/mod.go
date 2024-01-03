package go_cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/xhd2015/go-vendor-pack/go_cmd/model"
)

func ParseGoMod(dir string) (*model.GoMod, error) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer

	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Dir = dir
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("go list:%w %v", err, errBuf.String())
	}

	var goMod *model.GoMod
	err = json.Unmarshal(buf.Bytes(), &goMod)
	if err != nil {
		return nil, fmt.Errorf("decoding go.mod: %w", err)
	}
	return goMod, nil
}

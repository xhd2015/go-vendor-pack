package go_cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/xhd2015/go-vendor-pack/go_cmd/model"
)

func ParseGoMod(dirOrFile string) (*model.GoMod, error) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer

	stat, err := os.Stat(dirOrFile)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("go", "mod", "edit", "-json")
	if stat.IsDir() {
		cmd.Dir = dirOrFile
	} else {
		// actual file
		cmd.Args = append(cmd.Args, dirOrFile)
	}
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf

	err = cmd.Run()
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

func ParseGoModContent(content string) (*model.GoMod, error) {
	var goMod *model.GoMod
	err := GoModRead(content, func(goModFile string) error {
		var err error
		goMod, err = ParseGoMod(goModFile)
		return err
	})
	if err != nil {
		return nil, err
	}
	return goMod, nil
}

package go_cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func GoModRequire(goModFile string, module string, version string) error {
	if module == "" {
		return fmt.Errorf("requires module")
	}
	if version == "" {
		return fmt.Errorf("requires version")
	}
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := exec.Command("go", "mod", "edit", "-require="+module+"@"+version, goModFile)
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("replace: %w %s", err, errBuf.Bytes())
	}
	return nil
}

func GoModReplace(goModFile string, module string, replace string) error {
	if module == "" {
		return fmt.Errorf("requires module")
	}
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := exec.Command("go", "mod", "edit", "-replace="+module+"="+replace, goModFile)
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("replace: %w %s", err, errBuf.Bytes())
	}
	return nil
}

func GoModEdit(content string, edit func(goModFile string) error) (string, error) {
	var newContent []byte
	err := GoModRead(content, func(goModFile string) error {
		err := edit(goModFile)
		if err != nil {
			return err
		}
		newContent, err = ioutil.ReadFile(goModFile)
		return err
	})
	if err != nil {
		return "", err
	}
	return string(newContent), nil
}
func GoModRead(content string, read func(goModFile string) error) error {
	goModTmp, err := os.CreateTemp("", "go.mod")
	if err != nil {
		return err
	}

	_, err = goModTmp.WriteString(content)
	if err != nil {
		return err
	}
	err = goModTmp.Sync()
	if err != nil {
		return err
	}
	goModFile := goModTmp.Name()

	err = read(goModFile)
	if err != nil {
		return fmt.Errorf("go mod edit:%w", err)
	}
	return nil
}

package helper

import (
	"fmt"

	"github.com/xhd2015/go-inspect/sh"
	"github.com/xhd2015/go-vendor-pack/writefs"
)

func TruncateGoMod(goModFile string, module string, goVersionMajor int, goVersionMinor int) error {
	return TruncateGoModFS(writefs.SysFS{}, goModFile, module, goVersionMajor, goVersionMinor)
}
func TruncateGoModFS(wfs writefs.FS, goModFile string, module string, goVersionMajor int, goVersionMinor int) error {
	w, err := wfs.OpenFileWrite(goModFile)
	if err != nil {
		return fmt.Errorf("write missing go.mod for %s: %w", module, err)
	}
	defer w.Close()
	// always rewrite go.mod to clear transitive dependencies
	// this is an important trick to fool/hack the go build

	// check if go.mod already exists
	// _, statErr := os.Stat(tmpModuleGoMod)
	// if statErr != nil {
	// 	if !os.IsNotExist(statErr) {
	// 		return fmt.Errorf("check go.mod for %s: %w", module, statErr)
	// 	}
	// }
	// it's important to add go version declaration(e.g. go 1.18)
	// because go build system depends on this directive to
	// decide whether a feature can be used(like generic)
	_, err = w.Write([]byte(fmt.Sprintf("module %s\n\ngo %d.%d\n", module, goVersionMajor, goVersionMinor)))
	if err != nil {
		return fmt.Errorf("write missing go.mod for %s: %w", module, err)
	}
	return nil
}

func goModEditReplace(oldpath string, newPath string) string {
	return fmt.Sprintf("go mod edit -replace=%s=%s", sh.Quote(oldpath), sh.Quote(newPath))
}

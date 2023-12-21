package testdata

import (
	_ "github.com/xhd2015/go-inspect/sh"
	_ "golang.org/x/tools/cover"
)

//go:generate sh -c "go list -f '{{.ImportPath}} {{if .Module}}{{.Module.Version}}{{else}}{{end}}' -deps > go.mod.versions"
func Use() {

}

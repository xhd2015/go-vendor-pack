package model

import "github.com/xhd2015/go-vendor-pack/go_cmd/model"

type GoList struct {
	PackTimeUTC string
	GoMod       *model.GoMod
	Modules     []*Module
}

type Module struct {
	*model.ModulePublic
	Packages []*model.PackagePublic
}

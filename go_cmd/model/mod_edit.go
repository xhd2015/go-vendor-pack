package model

// see $GOROOT/src/cmd/go/internal/modcmd/edit.go
//    type Module struct

type GoMod struct {
	Module  ModPath
	Go      string // the go version
	Require []Require
	Exclude []GoModule
	Replace []Replace
	Retract []Retract
}

type GoModule struct {
	Path    string
	Version string
}

type ModPath struct {
	Path       string
	Deprecated string
}

type Require struct {
	Path     string
	Version  string
	Indirect bool
}

type Replace struct {
	Old GoModule
	New GoModule
}

type Retract struct {
	Low       string
	High      string
	Rationale string
}

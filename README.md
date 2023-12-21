# Usage

Pack source, and unpack in target.

## Pack

First, create a sub directory and add some dependency in go.mod.

Second, run `go mod tidy && go mod vendor` to let go update all depedencies.

Third, update `go.mod.vesion` with `go list -f '{{.ImportPath}} {{if .Module}}{{.Module.Version}}{{else}}{{end}}' -deps > go.mod.versions`.

In the last, add

```go
//go:embed pkg
var pack embed.FS
```

## Unpack

Will not override packages unless

```go
pack.Unpack(fs,"path/to/dst",&pack.UnpackOptions{})
```

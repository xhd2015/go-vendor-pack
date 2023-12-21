package unpack

import (
	"os"
	"os/exec"
	"testing"

	"github.com/xhd2015/go-inspect/sh"
)

func TestUnpack(t *testing.T) {
	os.RemoveAll("./testdata/target_patched")
	sh.RunBash([]string{
		"cp -R ./testdata/target ./testdata/target_patched",
	}, true)
	err := UnpackFromBase64Decode(testPack, "./testdata/target_patched", &Options{
		ForceUpgradeModules: map[string]bool{
			"github.com/xhd2015/go-inspect": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	MustBuild("./testdata/target_patched")
}

func MustBuild(dir string) {
	cmd := exec.Command("bash", "-xc", "go build ./...")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		panic("build failed")
	}
}

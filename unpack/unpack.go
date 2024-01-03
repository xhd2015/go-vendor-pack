package unpack

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"strings"

	"github.com/xhd2015/go-inspect/sh"
	"github.com/xhd2015/go-vendor-pack/go_info"
	pack_model "github.com/xhd2015/go-vendor-pack/pack/model"
	"github.com/xhd2015/go-vendor-pack/packfs"
	"github.com/xhd2015/go-vendor-pack/tar"
	"github.com/xhd2015/go-vendor-pack/unpack/helper"
)

type Options struct {
	NonVendorHostDir       string // when the target project is not vendor-style, this defines where to put modules.if empty, will create a temporary dir
	ForceUpgradeAllModules bool
	ForceUpgradeModules    map[string]bool
	ForceUpgradeModulePkgs map[string]map[string]bool // exmaple: {"a.b.c":{"d":true}}, NOTE: sub path should be relative
	// deprecated, use IgnoreUpdatingSums instead
	IgnoreSums         bool
	IgnoreUpdatingSums bool
	OptionalSumModules map[string]bool // some modules is replaced, they will not appear in go.sum
}

func NewTarFSWithBase64Decode(s string) (packfs.FS, error) {
	r := strings.NewReader(s)
	rawReader := base64.NewDecoder(base64.StdEncoding, r)
	return tar.NewTarFS(rawReader)
}

// UnpackFromBase64Decode will unpack files compressed in `s` into `dir`
func UnpackFromBase64Decode(s string, dir string, opts *Options) error {
	fs, err := NewTarFSWithBase64Decode(s)
	if err != nil {
		return err
	}
	return Unpack(fs, dir, opts)
}

func ReadGoList(fs packfs.FS) (*pack_model.GoList, error) {
	jsonData, err := fs.ReadFile("go.list.json")
	if err != nil {
		if !packfs.IsNotExists(err) {
			return nil, fmt.Errorf("missing go.list.json")
		}
		return nil, err
	}
	var goList *pack_model.GoList
	err = json.Unmarshal(jsonData, &goList)
	if err != nil {
		return nil, err
	}
	return goList, nil
}

func Unpack(fs packfs.FS, dir string, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	goVersion, err := go_info.GetGoVersionCached()
	if err != nil {
		return fmt.Errorf("get go version: %w", err)
	}
	forceUpgradeAll := opts.ForceUpgradeAllModules
	forceUpgradeModules := opts.ForceUpgradeModules
	versions, err := fs.ReadFile("go.mod.versions")
	if err != nil {
		return err
	}
	versionMapping := parseGoModVersions(string(versions))

	gomodWhitelistBytes, err := fs.ReadFile("go.mod.whitelist")
	if err != nil {
		if !packfs.IsNotExists(err) {
			return err
		}
	}
	gomodWhitelist := parseGoModWhitelist(string(gomodWhitelistBytes))

	goSums, err := fs.ReadFile("go.sum")
	if err != nil {
		return err
	}
	goSumMapping := parseGoSums(string(goSums))

	// check if has vendor dir
	vendorDir := path.Join(dir, "vendor")
	stat, statErr := os.Stat(vendorDir)
	var hasVendorDir bool
	if statErr != nil {
		if !os.IsNotExist(statErr) {
			return statErr
		}
	} else if stat.IsDir() {
		hasVendorDir = true
	}
	var tmpVendorDir string
	if !hasVendorDir {
		if opts.NonVendorHostDir != "" {
			tmpVendorDir = opts.NonVendorHostDir
		} else {
			var err error
			tmpVendorDir, err = os.MkdirTemp(os.TempDir(), "vendor")
			if err != nil {
				return err
			}
			log.Printf("creating temp non-vendor host dir: %s", tmpVendorDir)
		}
	}

	for module, version := range versionMapping {
		// skip non-whitelist
		if len(gomodWhitelist) > 0 && !gomodWhitelist[module] {
			continue
		}
		if version == "" {
			continue
		}
		// get sum
		optionalSum := opts.OptionalSumModules[module]
		sums := goSumMapping[module]
		if len(sums) == 0 && !optionalSum {
			return fmt.Errorf("module %s does not appear in go.sum, check if it is replaced, if so add it to OptionalSumModules", module)
		}
		targetDir := dir
		if !hasVendorDir {
			targetDir = tmpVendorDir
		}
		added, err := helper.AddVendor(targetDir, module, fs, forceUpgradeAll || forceUpgradeModules[module], opts.ForceUpgradeModulePkgs[module])
		if err != nil {
			return fmt.Errorf("unpacking %s: add vendor: %w", module, err)
		}
		if added && !(opts.IgnoreUpdatingSums || opts.IgnoreSums) {
			modSums := make([]string, 0, len(sums))
			for _, sum := range sums {
				modSums = append(modSums, fmt.Sprintf("%s %s", module, sum))
			}
			err := helper.AddVersionAndSum(dir, module, version, strings.Join(modSums, "\n"), "")
			if err != nil {
				return fmt.Errorf("unpacking %s: add dep %v", module, err)
			}
		}
		// update go.mod with replace, and add missing go.mod
		if !hasVendorDir {
			tmpModuleDir := path.Join(tmpVendorDir, "vendor", module)
			_, _, err := sh.RunBashWithOpts([]string{goModEditReplace(module, tmpModuleDir)}, sh.RunBashOptions{
				Verbose: true,
				FilterCmd: func(cmd *exec.Cmd) {
					cmd.Dir = dir
				},
			})
			if err != nil {
				return fmt.Errorf("replacing non-vendor module:%s %w", module, err)
			}

			err = helper.TruncateGoMod(path.Join(tmpModuleDir, "go.mod"), module, goVersion.Major, goVersion.Minor)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func goModEditReplace(oldpath string, newPath string) string {
	return fmt.Sprintf("go mod edit -replace=%s=%s", sh.Quote(oldpath), sh.Quote(newPath))
}

func parseGoModWhitelist(s string) map[string]bool {
	list := strings.Split(s, "\n")
	m := make(map[string]bool, len(list))
	for _, e := range list {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		m[e] = true
	}
	return m
}

// pkg->version
func parseGoModVersions(s string) map[string]string {
	list := strings.Split(s, "\n")
	m := make(map[string]string, len(list))
	for _, e := range list {
		sp := strings.SplitN(e, " ", 2)
		var pkg string
		var version string
		if len(sp) >= 1 {
			pkg = strings.TrimSpace(sp[0])
		}
		if len(sp) >= 2 {
			version = strings.TrimSpace(sp[1])
		}
		if pkg == "" {
			continue
		}
		m[pkg] = version
	}
	return m
}

// pkg->[]sums
func parseGoSums(s string) map[string][]string {
	list := strings.Split(s, "\n")
	m := make(map[string][]string, len(list))
	for _, e := range list {
		sp := strings.SplitN(e, " ", 2)
		var pkg string
		var sums string
		if len(sp) >= 1 {
			pkg = sp[0]
		}
		if len(sp) >= 2 {
			sums = sp[1]
		}
		if pkg == "" {
			continue
		}
		m[pkg] = append(m[pkg], sums)
	}
	return m
}

package pack

import (
	tarlib "archive/tar"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	pack_model "github.com/xhd2015/go-vendor-pack/pack/model"

	"github.com/xhd2015/go-inspect/sh"
	"github.com/xhd2015/go-vendor-pack/go_cmd"
	"github.com/xhd2015/go-vendor-pack/go_cmd/model"

	"github.com/xhd2015/go-vendor-pack/tar"
)

type Options struct {
	OutputDataFile            string
	RunGoModTidy              bool
	RunGoModVendor            bool
	ModuleWhitelist           map[string]bool
	RemoveNonWhitelistVendors bool
}

// f, err := os.OpenFile(dstFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
//
//	if err != nil {
//		return err
//	}
//
// defer f.Close()
func PackAsBase64ToCode(dir string, pkg string, varName string, dstFile string, opts *Options) error {
	data, err := PackAsBase64(dir, opts)
	if err != nil {
		return err
	}
	dataStr := string(data)
	code := fmt.Sprintf(`// Code generated by github.com/xhd2015/go-vendor-pack/cmd/go-pack. DO NOT EDIT.
package %s

var %s = "%s"
`, pkg, varName, dataStr)
	if opts != nil && opts.OutputDataFile != "" {
		err := ioutil.WriteFile(opts.OutputDataFile, data, 0755)
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(dstFile, []byte(code), 0755)
}

const FILE_GO_LIST_JSON = "go.list.json"

func PackAsBase64(dir string, opts *Options) ([]byte, error) {
	if opts == nil {
		opts = &Options{}
	}
	// run go mod tidy
	if opts.RunGoModTidy || opts.RunGoModVendor {
		var cmds []string
		if opts.RunGoModTidy {
			cmds = append(cmds, "go mod tidy")
		}
		if opts.RunGoModVendor {
			cmds = append(cmds, "go mod vendor")
		}
		_, _, err := sh.RunBashWithOpts(cmds, sh.RunBashOptions{
			FilterCmd: func(cmd *exec.Cmd) {
				cmd.Dir = dir
			},
		})
		if err != nil {
			return nil, err
		}
	}

	// TODO: deprecate go.mod.versions
	err := UpdateGoVersions(dir)
	if err != nil {
		return nil, err
	}
	modulesMapping, modules, err := GetGoListModules(dir)
	if err != nil {
		return nil, err
	}

	goMod, err := go_cmd.ParseGoMod(dir)
	if err != nil {
		return nil, err
	}

	// TODO: deprecate go.mod.whitelist
	goModWhitelist := path.Join(dir, "go.mod.whitelist")
	if len(opts.ModuleWhitelist) > 0 {
		whiteList := make([]string, 0, len(opts.ModuleWhitelist))
		oldModules := modules
		modules = make([]*pack_model.Module, 0, len(opts.ModuleWhitelist))
		for _, m := range oldModules {
			if !opts.ModuleWhitelist[m.Path] {
				continue
			}
			modules = append(modules, m)
		}
		for mod := range opts.ModuleWhitelist {
			whiteList = append(whiteList, mod)
			if modulesMapping[mod] == nil {
				return nil, fmt.Errorf("specified whitelist module does not exist: %s", mod)
			}
		}
		// sort whiteList
		sort.Strings(whiteList)
		err := ioutil.WriteFile(goModWhitelist, []byte(strings.Join(whiteList, "\n")), 0755)
		if err != nil {
			return nil, fmt.Errorf("writing go mod whitelist: %w", err)
		}
		if opts.RemoveNonWhitelistVendors {
			err := cleanVendors(dir, opts.ModuleWhitelist)
			if err != nil {
				return nil, fmt.Errorf("rm non whitelist: %w", err)
			}
		}
	} else {
		err := os.RemoveAll(goModWhitelist)
		if err != nil {
			return nil, fmt.Errorf("cleaning go mod whitelist: %w", err)
		}
	}

	h := md5.New()
	var buf bytes.Buffer
	writer := base64.NewEncoder(base64.StdEncoding, &buf)
	// NOTE: when pack, always set clearModTime to be true
	err = tarFilesAndVendors(dir, io.MultiWriter(writer, h), FILE_GO_LIST_JSON, opts.ModuleWhitelist, true /*clear mod time*/, func(twWriter *tarlib.Writer) error {
		digest := hex.EncodeToString(h.Sum(nil))

		var prevDigest string
		goListJSONFile := filepath.Join(dir, FILE_GO_LIST_JSON)
		origData, fileErr := ioutil.ReadFile(goListJSONFile)
		if fileErr != nil {
			if !os.IsNotExist(fileErr) {
				return fileErr
			}
		} else {
			type GoListDigest struct {
				Digest string
			}
			var goListDigest GoListDigest
			json.Unmarshal(origData, &goListDigest)
			prevDigest = goListDigest.Digest
		}
		goListData := origData
		if prevDigest == "" || prevDigest != digest {
			// write go.list.json
			goListJSON, err := json.Marshal(&pack_model.GoList{
				PackTimeUTC: time.Now().UTC().Format("2006-01-02 15:04:05"),
				Digest:      digest,
				GoMod:       goMod,
				Modules:     modules,
			})
			if err != nil {
				return err
			}
			goListData = goListJSON
			// update go.list.json only when anything changes
			err = ioutil.WriteFile(goListJSONFile, goListJSON, 0755)
			if err != nil {
				return fmt.Errorf("generating go.list.json: %w", err)
			}
		}
		return tar.TarAddFile(twWriter, FILE_GO_LIST_JSON, int64(len(goListData)), 0755, bytes.NewReader(goListData))
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
func cleanVendors(dir string, moduleWhitelist map[string]bool) error {
	vendorDir := path.Join(dir, "vendor")
	tmpVendorBakDir, err := os.MkdirTemp(os.TempDir(), "vendor_bak")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpVendorBakDir)

	vendorBakDir := path.Join(tmpVendorBakDir, "vendor")
	err = os.Rename(vendorDir, vendorBakDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(vendorDir, 0755)
	if err != nil {
		return err
	}

	err = os.Rename(path.Join(vendorBakDir, "modules.txt"), path.Join(vendorDir, "modules.txt"))
	if err != nil {
		return err
	}
	// sort whitelist so that longer name appears
	// first, to avoid rename conflict:
	//   rename /x/A/B /y/A/B
	//   rename /x/A /y/A  --> failed

	sortedWhitelist := make([]string, 0, len(moduleWhitelist))
	for mod := range moduleWhitelist {
		sortedWhitelist = append(sortedWhitelist, mod)
	}
	sort.Slice(sortedWhitelist, func(i, j int) bool {
		return len(sortedWhitelist[i]) < len(sortedWhitelist[j])
	})

	// copy vendor back
	for _, mod := range sortedWhitelist {
		srcDir := path.Join(vendorBakDir, mod)
		dstDir := path.Join(vendorDir, mod)
		err := os.MkdirAll(path.Dir(dstDir), 0755)
		if err != nil {
			return err
		}
		_, srcStatErr := os.Stat(srcDir)
		_, dstStatErr := os.Stat(dstDir)
		if os.IsNotExist(srcStatErr) && dstStatErr == nil {
			// if dstDir already exists, that means
			// parent directory already copied, we can skip
			continue
		}
		err = os.Rename(srcDir, dstDir)
		if err != nil {
			return fmt.Errorf("reserving module: %s %w", mod, err)
		}
	}
	return nil
}
func tarFilesAndVendors(dir string, writer io.Writer, excludeFile string, moduleWhitelist map[string]bool, clearModTime bool, afterWritten func(twWriter *tarlib.Writer) error) error {
	twWriter, flush, close := tar.WrapTarWriter(writer)
	defer close()

	// if no whitelist, pack all
	if len(moduleWhitelist) == 0 {
		err := tar.TarAppend(dir, twWriter, &tar.TarOptions{
			ClearModTime: clearModTime,
			ShouldInclude: func(relPath string, dir bool) bool {
				return relPath != excludeFile
			},
		})
		if err != nil {
			return err
		}
	} else {
		// otherwise, pack only whitelist
		// tar non-vendor first
		err := tar.TarAppend(dir, twWriter, &tar.TarOptions{
			ClearModTime: clearModTime,
			ShouldInclude: func(relPath string, dir bool) bool {
				return relPath != "vendor" && relPath != excludeFile
			},
		})
		if err != nil {
			return err
		}
		err = tar.TarAddDir(twWriter, "vendor", 0755)
		if err != nil {
			return err
		}
		// sort modules
		modulesSorted := make([]string, 0, len(moduleWhitelist))
		for mod := range moduleWhitelist {
			modulesSorted = append(modulesSorted, mod)
		}
		sort.Strings(modulesSorted)

		for _, mod := range modulesSorted {
			// add parent directories
			modList := strings.Split(mod, "/")
			for i := 1; i < len(modList); i++ {
				err := tar.TarAddDir(twWriter, path.Join("vendor", path.Join(modList[:i]...)), 0755)
				if err != nil {
					return err
				}
			}
			err = tar.TarAppend(path.Join(dir, "vendor", mod), twWriter, &tar.TarOptions{
				ClearModTime: clearModTime,
				WritePrefix:  path.Join("vendor", mod),
			})
			if err != nil {
				return err
			}
		}
	}
	if afterWritten != nil {
		flush()
		err := afterWritten(twWriter)
		if err != nil {
			return err
		}
	}
	return nil
}

func PackVendor(dir string, pkg string, w io.Writer) error {
	if pkg == "" {
		return fmt.Errorf("requires pkg")
	}
	return tar.Tar(filepath.Join(dir, "vendor", pkg), w, &tar.TarOptions{})
}

// # example:
// #   some.pkg v1.0.0/go.mod h1:Adsf=
func GetPkgSum(dir string, pkg string) ([]string, error) {
	if pkg == "" {
		return nil, fmt.Errorf("requires pkg")
	}
	content, err := ioutil.ReadFile(filepath.Join(dir, "go.sum"))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	var sums []string

	prefix := pkg + " "
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			sums = append(sums, line)
		}
	}
	return sums, nil
}

func GetPkgVersion(dir string, pkg string) (string, error) {
	if pkg == "" {
		return "", fmt.Errorf("requires pkg")
	}
	// go list -f {{.Module.Version}} "$pkg"
	res, _, err := sh.RunBashWithOpts([]string{
		fmt.Sprintf("go list -f {{.Module.Version}} %s", sh.Quote(pkg)),
	}, sh.RunBashOptions{NeedStdOut: true})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(res), nil
}

// can use this: go:generate cd pkg && go list -f '{{.ImportPath}} {{if .Module}}{{.Module.Version}}{{else}}{{end}}' -deps > go.mod.versions
// Deprecated
func UpdateGoVersions(dir string) error {
	// return sh.RunBash([]string{"go list -f '{{.ImportPath}} {{if .Module}}{{.Module.Version}}{{else}}{{end}}' -deps > go.mod.versions"}, false)
	var buf bytes.Buffer
	_, _, err := sh.RunBashWithOpts([]string{"go list -f '{{if .Module}}{{.Module.Path}} {{.Module.Version}}{{else}}{{end}}' -deps"}, sh.RunBashOptions{
		FilterCmd: func(cmd *exec.Cmd) {
			cmd.Dir = dir
			cmd.Stdout = &buf
		},
	})
	if err != nil {
		return err
	}
	lines := strings.Split(buf.String(), "\n")
	sort.Strings(lines)

	// invariant: [j,j) is uniq
	j := 0
	for i := 0; i < len(lines); i++ {
		if lines[i] != "" && (i == 0 || lines[i] != lines[i-1]) {
			lines[j] = lines[i]
			j++
		}
	}
	// sort uniq
	return ioutil.WriteFile(path.Join(dir, "go.mod.versions"), []byte(strings.Join(lines[:j], "\n")), 0755)
}

// can use this: go:generate cd pkg && go list -f '{{.ImportPath}} {{if .Module}}{{.Module.Version}}{{else}}{{end}}' -deps > go.mod.versions
// main module is excluded
func GetGoListModules(dir string) (map[string]*pack_model.Module, []*pack_model.Module, error) {
	pkgs, err := go_cmd.ListPackages(dir)
	if err != nil {
		return nil, nil, err
	}

	moduleMapping := make(map[string]*pack_model.Module)
	var modules []*pack_model.Module
	for _, pkg := range pkgs {
		// skip standard packages
		if pkg.Standard {
			continue
		}
		if pkg.Module == nil || pkg.Module.Path == "" {
			return nil, nil, fmt.Errorf("non module package: %v", pkg.ImportPath)
		}
		mod, ok := moduleMapping[pkg.Module.Path]
		if !ok {
			mod = &pack_model.Module{
				ModulePublic: &model.ModulePublic{
					Path:      pkg.Module.Path,
					Version:   pkg.Module.Version,
					GoVersion: pkg.Module.GoVersion,
					Indirect:  pkg.Module.Indirect,
				},
			}
			modules = append(modules, mod)
			moduleMapping[pkg.Module.Path] = mod
		}
		mod.Packages = append(mod.Packages, &model.PackagePublic{
			ImportPath: pkg.ImportPath,
			Name:       pkg.Name,
		})
	}
	return moduleMapping, modules, nil
}

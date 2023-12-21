package go_info

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/xhd2015/go-inspect/inspect/util"
)

// goDownloaderRepo:
//    example branch: go1.18/{OS}/{Arch}
//    expect files:  any-name.tar.gz  --> the only file with .tar.gz is considered go's release
func SetupGoWithSrc(goDownloaderRepo string) error {
	ok, err := GorootContainsSource()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	version, err := GetGoVersion()
	if err != nil {
		return err
	}
	newGoroot, err := cloneGo(version, goDownloaderRepo)

	if err != nil {
		return err
	}
	log.Printf("use downloaded GOROOT: %s", newGoroot)
	os.Setenv("GOROOT", newGoroot)
	os.Setenv("PATH", fmt.Sprintf("%s/bin:%s", newGoroot, os.Getenv("PATH")))
	return nil
}

func GorootContainsSource() (bool, error) {
	goRoot, err := util.ComputeGOROOT()
	if err != nil {
		return false, err
	}
	runtimeDir := path.Join(goRoot, "src/runtime")
	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go") {
			return true, nil
		}
	}
	return false, nil
}

type GoVersion struct {
	Major int // 1
	Minor int // 17
	Patch int // 5

	OS   string
	Arch string
}

func GetGoVersion() (*GoVersion, error) {
	// go version go1.17.5 darwin/amd64
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return nil, fmt.Errorf("cannot get go version")
	}
	outTrim := strings.TrimSuffix(string(out), "\n")
	return parseGoVersion(outTrim)
}

var gooVersionOnce sync.Once
var goVersion *GoVersion
var goVersionErr error

func GetGoVersionCached() (*GoVersion, error) {
	gooVersionOnce.Do(func() {
		goVersion, goVersionErr = GetGoVersion()
	})
	return goVersion, goVersionErr
}

const goVersionPrefix = "go version "

func parseGoVersion(s string) (*GoVersion, error) {
	if !strings.HasPrefix(s, goVersionPrefix) {
		return nil, fmt.Errorf("unrecognized version, expect prefix '%s': %s", goVersionPrefix, s)
	}
	s = s[len(goVersionPrefix):]
	if !strings.HasPrefix(s, "go") {
		return nil, fmt.Errorf("unrecognized version, expect pattern 'go1.x.y': %s", s)
	}
	s = s[len("go"):]

	spaceIdx := strings.Index(s, " ")
	if spaceIdx < 0 {
		return nil, fmt.Errorf("unrecognized version, expect space after 'go1.x.y': %s", s)
	}
	version := s[:spaceIdx]
	osArch := s[spaceIdx+1:]

	res := &GoVersion{}
	verList := strings.Split(version, ".")
	for i := 0; i < 3; i++ {
		if i < len(verList) {
			verInt, err := strconv.ParseInt(verList[i], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("unrecognized version, expect number, found: %s", version)
			}
			switch i {
			case 0:
				res.Major = int(verInt)
			case 1:
				res.Minor = int(verInt)
			case 2:
				res.Patch = int(verInt)
			}
		}
	}
	slashIdx := strings.Index(osArch, "/")
	if slashIdx < 0 {
		return nil, fmt.Errorf("unrecognized version, expect os/arch: %s", osArch)
	}
	res.OS = osArch[:slashIdx]
	res.Arch = osArch[slashIdx+1:]
	return res, nil
}

func cloneGo(version *GoVersion, goDownloaderRepo string) (goRoot string, err error) {
	// git clone --depth=1 --branch=go1.13/linux/amd64 --single-branch "https://gitlab:TOKEN@github.com/some/go-downloader
	tmpDir, err := os.MkdirTemp(os.TempDir(), "go-downloader")
	if err != nil {
		return "", err
	}
	cloneDir := path.Join(tmpDir, "go-downloader")
	cmd := exec.Command("git", "clone", "--depth=1", fmt.Sprintf("--branch=go%d.%d/%s/%s", version.Major, version.Minor, version.OS, version.Arch), "--single-branch", goDownloaderRepo, cloneDir)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(cloneDir)
	if err != nil {
		return "", err
	}
	var gzFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tar.gz") {
			gzFile = entry.Name()
			break
		}
	}
	if gzFile == "" {
		return "", fmt.Errorf("no tar.gz file for downloaded go")
	}
	// tar -xzf go.tar.gz
	unizpCmd := exec.Command("tar", "-C", cloneDir, "-xzf", path.Join(cloneDir, gzFile))
	unizpCmd.Stderr = os.Stderr
	unizpCmd.Stdout = os.Stdout
	err = unizpCmd.Run()

	if err != nil {
		return "", err
	}
	return path.Join(cloneDir, "go"), nil
}

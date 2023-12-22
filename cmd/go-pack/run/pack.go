package run

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/go-vendor-pack/pack"
	"github.com/xhd2015/go-vendor-pack/prog"
)

// example:
//   $ pack
//   $ unpack

type Prog struct {

	// dir string, pkg string, varName string, dstFile string

	Dir                       string `prog:"dir '' working directory"`
	Pkg                       string `prog:"pkg '' package name"`
	Var                       string `prog:"var '' var name"`
	Output                    string `prog:"o '' output file"`
	OutputDataFile            string `prog:"output-data-file '' output data file"`
	RunGoModTidy              bool   `prog:"run-go-mod-tidy false run go mod tidy before pack"`
	RunGoModVendor            bool   `prog:"run-go-mod-vendor false run go mod vendor before pack"`
	ModuleWhitelist           string `prog:"module-whitelist '' module whitelist,separated by comma"`
	RemoveNonWhitelistVendors bool   `prog:"rm-non-whitelist-vendors false remove non-whitelist vendors"`

	// for unpack
	InputDataFile    string `prog:"input-data-file '' input data file"`
	UnpackIgnoreSums bool   `prog:"unpack-ignore-sums false ignore sums when unpack"`
}

var progArgs Prog

var commands = map[string]func(comm string, args []string, extraArgs []string){
	"help":     help,
	"version":  version,
	"pack":     packCmd,
	"unpack":   unpackCmd,
	"show-env": showEnv,
}

func Main() {
	prog.Run(&progArgs, &prog.RunOptions{
		Usage: usage,
		AfterFlagParse: func() {
		},
		Commands: commands,
		Default:  defaultCommand,
	})
}

func help(commd string, args []string, extraArgs []string) {
	flag.Usage()
	os.Exit(0)
}

func packCmd(commd string, args []string, extraArgs []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "requires dir\n")
		os.Exit(1)
	}
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "requires only 1 dir\n")
		os.Exit(1)
	}
	dir := args[0]
	if progArgs.Pkg == "" {
		fmt.Fprintf(os.Stderr, "requires pkg\n")
		os.Exit(1)
	}
	if progArgs.Var == "" {
		fmt.Fprintf(os.Stderr, "requires var\n")
		os.Exit(1)
	}
	if progArgs.Output == "" {
		fmt.Fprintf(os.Stderr, "requires output\n")
		os.Exit(1)
	}
	var moduleWhitelist map[string]bool
	if progArgs.ModuleWhitelist != "" {
		modules := strings.Split(progArgs.ModuleWhitelist, ",")
		moduleWhitelist = make(map[string]bool, len(modules))
		for _, mod := range modules {
			mod = strings.TrimSpace(mod)
			if mod != "" {
				moduleWhitelist[mod] = true
			}
		}
	}
	err := pack.PackAsBase64ToCode(dir, progArgs.Pkg, progArgs.Var, progArgs.Output, &pack.Options{
		OutputDataFile:            progArgs.OutputDataFile,
		RunGoModTidy:              progArgs.RunGoModTidy,
		RunGoModVendor:            progArgs.RunGoModVendor,
		ModuleWhitelist:           moduleWhitelist,
		RemoveNonWhitelistVendors: progArgs.RemoveNonWhitelistVendors,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func showEnv(commd string, args []string, extraArgs []string) {
	fmt.Println(strings.Join(os.Environ(), "\n"))
}
func version(commd string, args []string, extraArgs []string) {
	fmt.Println("v1.0.0")
}

func defaultCommand(commd string, args []string, extraArgs []string) {
	if commd == "" {
		fmt.Printf("requries cmd: pack,unpack,help\n")
	} else {
		fmt.Printf("unknown cmd:%s\n", commd)
	}
	flag.Usage()
	os.Exit(1)
}

func usage(defaultUsage func()) func() {
	return func() {
		fmt.Sprintf(strings.Join([]string{
			"supported commands: pack,unpack\n",
			"    pack DIR -dst X\n",
			"        build the package with generated mock stubs,default output is exec.bin or debug.bin if -debug\n",
			"    unpack DIR[--] [EXEC_ARGS]\n",
			"    help\n",
			"        show help message\n",
		}, "\n"))
		defaultUsage()
		fmt.Sprintf(strings.Join([]string{
			"examples:\n",
		}, "\n"))
	}
}

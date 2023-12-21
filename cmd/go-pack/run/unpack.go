package run

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/xhd2015/go-vendor-pack/unpack"
)

func unpackCmd(commd string, args []string, extraArgs []string) {
	if len(args) == 0 || args[0] == "" {
		fmt.Fprintf(os.Stderr, "requires dir\n")
		os.Exit(1)
	}
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "requires only 1 dir\n")
		os.Exit(1)
	}
	dir := args[0]
	inputFile := progArgs.InputDataFile
	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "requires -input-data-file")
		os.Exit(1)

	}
	inputData, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	err = unpack.UnpackFromBase64Decode(string(inputData), dir, &unpack.Options{
		IgnoreSums: progArgs.UnpackIgnoreSums,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

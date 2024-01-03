package model

import (
	"encoding/json"
	"strings"
)

// see $GOROOT/src/cmd/go/internal/load/pkg.go
//     type PackagePublic struct {...}

type PackagePublic struct {
	// Note: These fields are part of the go command's public API.
	// See list.go. It is okay to add fields, but not to change or
	// remove existing ones. Keep in sync with list.go
	Dir           string        `json:",omitempty"` // directory containing package sources
	ImportPath    string        `json:",omitempty"` // import path of package in dir
	ImportComment string        `json:",omitempty"` // path in import comment on package statement
	Name          string        `json:",omitempty"` // package name
	Doc           string        `json:",omitempty"` // package documentation string
	Target        string        `json:",omitempty"` // installed target for this package (may be executable)
	Shlib         string        `json:",omitempty"` // the shared library that contains this package (only set when -linkshared)
	Root          string        `json:",omitempty"` // Go root, Go path dir, or module root dir containing this package
	ConflictDir   string        `json:",omitempty"` // Dir is hidden by this other directory
	ForTest       string        `json:",omitempty"` // package is only for use in named test
	Export        string        `json:",omitempty"` // file containing export data (set by go list -export)
	BuildID       string        `json:",omitempty"` // build ID of the compiled package (set by go list -export)
	Module        *ModulePublic `json:",omitempty"` // info about package's module, if any
	Match         []string      `json:",omitempty"` // command-line patterns matching this package
	Goroot        bool          `json:",omitempty"` // is this package found in the Go root?
	Standard      bool          `json:",omitempty"` // is this package part of the standard Go library?
	DepOnly       bool          `json:",omitempty"` // package is only as a dependency, not explicitly listed
	BinaryOnly    bool          `json:",omitempty"` // package cannot be recompiled
	Incomplete    bool          `json:",omitempty"` // was there an error loading this package or dependencies?

	// Stale and StaleReason remain here *only* for the list command.
	// They are only initialized in preparation for list execution.
	// The regular build determines staleness on the fly during action execution.
	Stale       bool   `json:",omitempty"` // would 'go install' do anything for this package?
	StaleReason string `json:",omitempty"` // why is Stale true?

	// Source files
	// If you add to this list you MUST add to p.AllFiles (below) too.
	// Otherwise file name security lists will not apply to any new additions.
	GoFiles           []string `json:",omitempty"` // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles          []string `json:",omitempty"` // .go source files that import "C"
	CompiledGoFiles   []string `json:",omitempty"` // .go output from running cgo on CgoFiles
	IgnoredGoFiles    []string `json:",omitempty"` // .go source files ignored due to build constraints
	InvalidGoFiles    []string `json:",omitempty"` // .go source files with detected problems (parse error, wrong package name, and so on)
	IgnoredOtherFiles []string `json:",omitempty"` // non-.go source files ignored due to build constraints
	CFiles            []string `json:",omitempty"` // .c source files
	CXXFiles          []string `json:",omitempty"` // .cc, .cpp and .cxx source files
	MFiles            []string `json:",omitempty"` // .m source files
	HFiles            []string `json:",omitempty"` // .h, .hh, .hpp and .hxx source files
	FFiles            []string `json:",omitempty"` // .f, .F, .for and .f90 Fortran source files
	SFiles            []string `json:",omitempty"` // .s source files
	SwigFiles         []string `json:",omitempty"` // .swig files
	SwigCXXFiles      []string `json:",omitempty"` // .swigcxx files
	SysoFiles         []string `json:",omitempty"` // .syso system object files added to package

	NotLoadedDeps []string `json:",omitempty"`

	// Embedded files
	EmbedPatterns []string `json:",omitempty"` // //go:embed patterns
	EmbedFiles    []string `json:",omitempty"` // files matched by EmbedPatterns

	// Cgo directives
	CgoCFLAGS    []string `json:",omitempty"` // cgo: flags for C compiler
	CgoCPPFLAGS  []string `json:",omitempty"` // cgo: flags for C preprocessor
	CgoCXXFLAGS  []string `json:",omitempty"` // cgo: flags for C++ compiler
	CgoFFLAGS    []string `json:",omitempty"` // cgo: flags for Fortran compiler
	CgoLDFLAGS   []string `json:",omitempty"` // cgo: flags for linker
	CgoPkgConfig []string `json:",omitempty"` // cgo: pkg-config names

	// Dependency information
	Imports   []string          `json:",omitempty"` // import paths used by this package
	ImportMap map[string]string `json:",omitempty"` // map from source import to ImportPath (identity entries omitted)
	Deps      []string          `json:",omitempty"` // all (recursively) imported dependencies

	// Error information
	// Incomplete is above, packed into the other bools
	Error      *PackageError   `json:",omitempty"` // error loading this package (not dependencies)
	DepsErrors []*PackageError `json:",omitempty"` // errors loading dependencies

	// Test information
	// If you add to this list you MUST add to p.AllFiles (below) too.
	// Otherwise file name security lists will not apply to any new additions.
	TestGoFiles        []string `json:",omitempty"` // _test.go files in package
	TestImports        []string `json:",omitempty"` // imports from TestGoFiles
	TestEmbedPatterns  []string `json:",omitempty"` // //go:embed patterns
	TestEmbedFiles     []string `json:",omitempty"` // files matched by TestEmbedPatterns
	XTestGoFiles       []string `json:",omitempty"` // _test.go files outside package
	XTestImports       []string `json:",omitempty"` // imports from XTestGoFiles
	XTestEmbedPatterns []string `json:",omitempty"` // //go:embed patterns
	XTestEmbedFiles    []string `json:",omitempty"` // files matched by XTestEmbedPatterns
}

// A PackageError describes an error loading information about a package.
type PackageError struct {
	ImportStack      []string // shortest path from package named on command line to this one
	Pos              string   // position of error
	Err              error    // the error itself
	IsImportCycle    bool     // the error is an import cycle
	Hard             bool     // whether the error is soft or hard; soft errors are ignored in some places
	alwaysPrintStack bool     // whether to always print the ImportStack
}

func (p *PackageError) Error() string {
	// TODO(#43696): decide when to print the stack or the position based on
	// the error type and whether the package is in the main module.
	// Document the rationale.
	if p.Pos != "" && (len(p.ImportStack) == 0 || !p.alwaysPrintStack) {
		// Omit import stack. The full path to the file where the error
		// is the most important thing.
		return p.Pos + ": " + p.Err.Error()
	}

	// If the error is an ImportPathError, and the last path on the stack appears
	// in the error message, omit that path from the stack to avoid repetition.
	// If an ImportPathError wraps another ImportPathError that matches the
	// last path on the stack, we don't omit the path. An error like
	// "package A imports B: error loading C caused by B" would not be clearer
	// if "imports B" were omitted.
	if len(p.ImportStack) == 0 {
		return p.Err.Error()
	}
	var optpos string
	if p.Pos != "" {
		optpos = "\n\t" + p.Pos
	}
	return "package " + strings.Join(p.ImportStack, "\n\timports ") + optpos + ": " + p.Err.Error()
}

func (p *PackageError) Unwrap() error { return p.Err }

// PackageError implements MarshalJSON so that Err is marshaled as a string
// and non-essential fields are omitted.
func (p *PackageError) MarshalJSON() ([]byte, error) {
	perr := struct {
		ImportStack []string
		Pos         string
		Err         string
	}{p.ImportStack, p.Pos, p.Err.Error()}
	return json.Marshal(perr)
}

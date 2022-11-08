package pbthrift

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"strconv"

	"github.com/kevinzfb/protobuf-thrift/utils/logger"
)

const (
	TASK_FILE_PROTO2THRIFT = iota + 1
	TASK_FILE_THRIFT2PROTO
	TASK_CONTENT_PROTO2THRIFT
	TASK_CONTENT_THRIFT2PROTO
)

type Runner struct {
	Config *RunnerConfig
}

type RunnerConfig struct {
	Pipe       bool // return the result from Generator instead of printing to os.Stdout or filesystem
	RawContent string
	InputPath  string // absolute path for input idl file
	OutputDir  string // absolute path for output dir
	Task       int
	Recursive  bool // recursive parse file with imported files

	UseSpaceIndent bool
	IndentSpace    string
	FieldCase      string
	NameCase       string

	// pb config
	Syntax int // 2 or 3

	// [kevinzfb] forked config options
	AddUnknownToEnums  bool            // best practice for Thrift
	Namespace          string          // allows for specifying a different namespace
	NamespaceLangs     []string        // list of languages for namespace declaration
	EnumCase           string
}

func NewRunner() (res *Runner, err error) {
	var rawContent, inputPath, outputDir, taskType, useSpaceIndent, indentSpace string
	var nameCase, fieldCase string
	var syntaxStr, recursiveStr string

	// [kevinzfb] forked params
	var addUnknownToEnums, namespace, nsLangs, enumCase string

	// flags declaration using flag package
	flag.StringVar(&taskType, "t", "", "proto => thrift or thrift => proto, valid values proto2thrift and thrift2proto")
	flag.StringVar(&inputPath, "i", "", "The idl's file path or directory, if is a directory, it will iterate all idl files")
	flag.StringVar(&outputDir, "o", "", "The output idl dir path")
	flag.StringVar(&recursiveStr, "r", "0", "Recursive parse file with imported files")
	flag.StringVar(&useSpaceIndent, "use-space-indent", "0", "Use space for indent rather than tab")
	flag.StringVar(&indentSpace, "indent-space", "4", "The space count for each indent")
	flag.StringVar(&fieldCase, "field-case", "camelCase", "Text case for message or struct field, available options: camelCase, snakeCase, kababCase, pascalCase, screamingSnakeCase")
	flag.StringVar(&nameCase, "name-case", "camelCase", "Text case for enum and message or struct name, available options: camelCase, snakeCase, kababCase, pascalCase, screamingSnakeCase")
	flag.StringVar(&syntaxStr, "syntax", "3", "Syntax for generated protobuf idl")

	// [kevinzfb] added flags
	flag.StringVar(&addUnknownToEnums, "add-unknown-to-enums", "", "Add an UNKNOWN entry with value 0 at the beginning of every enum (proto2thrift only)")
	flag.StringVar(&namespace, "namespace", "", "Overrides namespace in each file with the specified namespace (proto2thrift only)")
	flag.StringVar(&nsLangs, "namespace-languages", "*", "Languages to be used in namespace declarations (proto2thrift only)")
	flag.StringVar(&enumCase, "enum-case", "screamingSnakeCase", "Text case for enum field, available options: camelCase, snakeCase, kababCase, pascalCase, screamingSnakeCase (proto2thrift only for now)")

	flag.Parse() // after declaring flags we need to call it

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	// validate cli params
	ValidateTaskType(taskType)
	ValidateIndentSpace(indentSpace)
	syntax := ValidateSyntax(syntaxStr)
	recursive := ValidateRecursive(recursiveStr)
	spaceIndent := useSpaceIndent == "1"
	namespaceLangs := strings.Split(nsLangs, ",")
	unknownToEnums := addUnknownToEnums != ""
	var task int
	if taskType == "proto2thrift" {
		if inputPath != "" {
			task = TASK_FILE_PROTO2THRIFT
		} else {
			task = TASK_CONTENT_PROTO2THRIFT
		}
	} else if taskType == "thrift2proto" {
		if inputPath != "" {
			task = TASK_FILE_THRIFT2PROTO
		} else {
			task = TASK_CONTENT_THRIFT2PROTO
		}
	}
	if task == TASK_FILE_PROTO2THRIFT || task == TASK_FILE_THRIFT2PROTO {
		inputPath, outputDir = ValidateInputAndOutput(inputPath, outputDir)
	}

	// read rawContent from stdin directly
	if task == TASK_CONTENT_PROTO2THRIFT || task == TASK_CONTENT_THRIFT2PROTO {
		logger.Info("Paste your original idl here, then press Ctrl+D to continue =>")

		var bytes []byte
		bytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			logger.Errorf("read data from stdin error %v", err)
			return
		}

		logger.Info("Converting...")
		rawContent = string(bytes)
	}

	config := &RunnerConfig{
		RawContent:     rawContent,
		InputPath:      inputPath,
		OutputDir:      outputDir,
		UseSpaceIndent: spaceIndent,
		IndentSpace:    indentSpace,
		FieldCase:      fieldCase,
		NameCase:       nameCase,
		Task:           task,
		Syntax:         syntax,
		Recursive:      recursive,

		// [kevinzfb] forked configs
		AddUnknownToEnums: unknownToEnums,
		Namespace:         namespace,
		NamespaceLangs:    namespaceLangs,
		EnumCase:          enumCase,
	}
	res = &Runner{
		Config: config,
	}
	return
}

func (r *Runner) Run() (err error) {
	var generator Generator
	generator, err = NewGenerator(r.Config)
	if err != nil {
		return
	}
	err = generator.Generate()
	return
}

func (r *Runner) Pipe() (res []byte, err error) {
	var generator Generator
	generator, err = NewGenerator(r.Config)
	if err != nil {
		return
	}
	res, err = generator.Pipe()
	return
}

func ValidateTaskType(taskType string) {
	if taskType != "proto2thrift" && taskType != "thrift2proto" {
		logger.Fatal("You must specify which task you want to run, proto2thrift or thrift2proto.")
	}
}

func ValidateInputAndOutput(inputPath string, outputDir string) (inputAbs string, outputAbs string) {
	if inputPath != "" && outputDir == "" {
		logger.Fatal("You must specify the output path.")
	}

	if filepath.IsAbs(inputPath) {
		inputAbs = inputPath
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			logger.Fatal(err)
			return
		}
		inputAbs = filepath.Join(cwd, inputPath)
	}

	if filepath.IsAbs(outputDir) {
		outputAbs = outputDir
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			logger.Fatal(err)
			return
		}
		outputAbs = filepath.Join(cwd, outputDir)
	}
	return
}

func ValidateIndentSpace(indentSpace string) {
	_, err := strconv.Atoi(indentSpace)
	if err != nil {
		logger.Fatalf("Invalid indent-space option %v", indentSpace)
	}
}

func ValidateSyntax(syntaxStr string) (res int) {
	var err error
	if res, err = strconv.Atoi(syntaxStr); err != nil {
		logger.Fatalf("Invalid syntax option %v", syntaxStr)
	}
	return
}

func ValidateRecursive(recursiveStr string) (res bool) {
	var err error
	var resInt int
	if resInt, err = strconv.Atoi(recursiveStr); err != nil {
		logger.Fatalf("Invalid recursive option %v", recursiveStr)
	} else {
		if resInt == 1 {
			res = true
		} else {
			res = false
		}
	}
	return
}

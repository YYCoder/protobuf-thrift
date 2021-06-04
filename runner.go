package main

import (
	"flag"
	"io"
	"io/fs"
	"os"
	"strconv"

	"github.com/protobuf-thrift/utils/logger"
)

const (
	TASK_FILE_PROTO2THRIFT = iota + 1
	TASK_FILE_THRIFT2PROTO
	TASK_CONTENT_PROTO2THRIFT
	TASK_CONTENT_THRIFT2PROTO
)

type Runner struct {
	config *RunnerConfig
}

type RunnerConfig struct {
	RawContent string
	InputPath  string
	OutputPath string
	Task       int

	UseSpaceIndent bool
	IndentSpace    string
	FieldCase      string
	NameCase       string
}

func NewRunner() (res *Runner, err error) {
	var rawContent, inputPath, outputPath, taskType, useSpaceIndent, indentSpace string
	var nameCase, fieldCase string

	// flags declaration using flag package
	flag.StringVar(&taskType, "t", "", "proto => thrift or thrift => proto, valid values proto2thrift and thrift2proto")
	flag.StringVar(&inputPath, "i", "", "The idl's file path or directory, if is a directory, it will iterate all idl files")
	flag.StringVar(&outputPath, "o", "", "The output idl file path")
	flag.StringVar(&useSpaceIndent, "use-space-indent", "0", "Use space for indent rather than tab")
	flag.StringVar(&indentSpace, "indent-space", "4", "The space count for each indent")
	flag.StringVar(&fieldCase, "field-case", "camelCase", "Text case for enum field and message or struct field, available options: camelCase, snakeCase, kababCase, pascalCase, screamingSnakeCase")
	flag.StringVar(&nameCase, "name-case", "camelCase", "Text case for enum and message or struct name, available options: camelCase, snakeCase, kababCase, pascalCase, screamingSnakeCase")

	flag.Parse() // after declaring flags we need to call it

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	// check if cli params match
	if taskType != "proto2thrift" && taskType != "thrift2proto" {
		logger.Fatal("You must specify which task you want to run, proto2thrift or thrift2proto.")
	} else if inputPath != "" && outputPath == "" {
		logger.Fatal("You must specify the output path.")
	}

	_, err = strconv.Atoi(indentSpace)
	if err != nil {
		logger.Fatalf("Invalid indent-space option %v", indentSpace)
	}

	var task int
	spaceIndent := useSpaceIndent == "1"
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
		OutputPath:     outputPath,
		UseSpaceIndent: spaceIndent,
		IndentSpace:    indentSpace,
		FieldCase:      fieldCase,
		NameCase:       nameCase,
		Task:           task,
	}
	res = &Runner{
		config: config,
	}
	return
}

func (r *Runner) Run() (err error) {
	switch r.config.Task {
	case TASK_CONTENT_PROTO2THRIFT:
		var generator *ThriftGenerator
		generator, err = NewThriftGenerator(r.config, "")
		if err != nil {
			return err
		}
		err = generator.Generate()

	case TASK_FILE_PROTO2THRIFT:
		var file *os.File
		file, err = os.Open(r.config.InputPath)
		if err != nil {
			return err
		}
		defer file.Close()

		var stat fs.FileInfo
		stat, err = file.Stat()
		if err != nil {
			return err
		}

		if stat.IsDir() {
			// TODO: recursivly generate all idl files
		} else {
			var generator *ThriftGenerator
			generator, err = NewThriftGenerator(r.config, r.config.InputPath)
			if err != nil {
				return err
			}
			err = generator.Generate()
		}

	case TASK_CONTENT_THRIFT2PROTO:
		var generator *ProtoGenerator
		generator, err = NewProtoGenerator(r.config, "")
		if err != nil {
			return err
		}
		err = generator.Generate()

	case TASK_FILE_THRIFT2PROTO:
		var file *os.File
		file, err = os.Open(r.config.InputPath)
		if err != nil {
			return err
		}
		defer file.Close()

		var stat fs.FileInfo
		stat, err = file.Stat()
		if err != nil {
			return err
		}

		if stat.IsDir() {
			// TODO: recursivly generate all idl files
		} else {
			var generator *ProtoGenerator
			generator, err = NewProtoGenerator(r.config, r.config.InputPath)
			if err != nil {
				return err
			}
			err = generator.Generate()
		}
	}
	return
}

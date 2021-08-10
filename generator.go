package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/YYCoder/protobuf-thrift/utils/logger"
)

// Represent a file need to be converted, including current absolute path and absolute converted output path
type FileInfo struct {
	absPath    string // absolute path for this included file
	outputPath string // absolute path for the output file
}

// Generator for each idl file
type SubGenerator interface {
	Parse() (newFiles []FileInfo, err error) // return relative file path to parsed file
	Sink() (err error)
	FilePath() (res string)
}

// Main generator for all idl files, responsible for initialize all SubGenerator for each file
type Generator interface {
	Generate() (err error)
}

func NewGenerator(conf *RunnerConfig) (res Generator, err error) {
	gen := &generator{
		conf:            conf,
		filesStack:      []FileInfo{},
		subGeneratorMap: make(map[string]SubGenerator),
	}

	if conf.Task == TASK_CONTENT_PROTO2THRIFT || conf.Task == TASK_CONTENT_THRIFT2PROTO {
		gen.initSubGeneratorForRawContent()
	} else {
		_, filename := filepath.Split(conf.InputPath)

		gen.initSubGenerator([]FileInfo{
			{
				absPath:    gen.conf.InputPath,
				outputPath: filepath.Join(gen.conf.OutputDir, gen.replaceExt(filename)),
			},
		})
	}

	res = gen
	return
}

type generator struct {
	conf *RunnerConfig
	// stack is empty when there is no file need to be generated, eash one is an abs file path. For rawContent task, use a default name
	filesStack      []FileInfo
	subGeneratorMap map[string]SubGenerator
}

func (g *generator) Generate() (err error) {
	for len(g.filesStack) > 0 {
		var lastFilePath FileInfo
		lastFilePath, g.filesStack = g.filesStack[len(g.filesStack)-1], g.filesStack[:len(g.filesStack)-1]
		sub, ok := g.subGeneratorMap[lastFilePath.absPath]
		if !ok {
			logger.Fatalf("Can't find file %v's sub generator", lastFilePath)
			return
		}

		var newFiles []FileInfo
		if newFiles, err = sub.Parse(); err != nil {
			logger.Fatalf("Error occurred when parsing file %v", sub.FilePath())
			return
		} else if len(newFiles) > 0 && g.conf.Recursive {
			err = g.initSubGenerator(newFiles)
			if err != nil {
				logger.Fatalf("Error occurred when initSubGenerator file %v, %v", sub.FilePath(), err)
				return
			}
		}
	}

	for _, sub := range g.subGeneratorMap {
		if err = sub.Sink(); err != nil {
			logger.Fatalf("Error occurred when generating file %v", sub.FilePath(), err)
			return
		}
	}
	return
}

func (g *generator) absPathIsIdl(absPath string) (res bool, err error) {
	suffix := ""
	if g.conf.Task == TASK_CONTENT_PROTO2THRIFT || g.conf.Task == TASK_FILE_PROTO2THRIFT {
		suffix = ".proto"
	} else {
		suffix = ".thrift"
	}
	ext := filepath.Ext(absPath)
	res = ext == suffix
	return
}

func (g *generator) replaceExt(filename string) (res string) {
	oldExt, newExt := "", ""
	if g.conf.Task == TASK_FILE_PROTO2THRIFT {
		oldExt = ".proto"
		newExt = ".thrift"
	} else {
		oldExt = ".thrift"
		newExt = ".proto"
	}
	res = strings.ReplaceAll(filename, oldExt, newExt)
	return
}

// get all absolute file paths from single dir
func (g *generator) getAllFileFromDir(root string) (res []FileInfo, err error) {
	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		var absPath string
		if filepath.IsAbs(path) {
			absPath = path
		} else {
			absPath = filepath.Join(root, path)
		}
		isIdl, _ := g.absPathIsIdl(absPath)
		if isIdl {
			relPath, err := filepath.Rel(root, absPath)
			if err != nil {
				logger.Errorf("filepath.Rel %v %v, err %v", root, path, err)
				return nil
			}
			newFile := FileInfo{
				absPath:    absPath,
				outputPath: filepath.Join(g.conf.OutputDir, g.replaceExt(relPath)),
			}
			res = append(res, newFile)
		}
		return nil
	})
	return
}

func (g *generator) initSubGenerator(fileInfos []FileInfo) (err error) {
	logger.Infof("initSubGenerator start: %+v", fileInfos)

	files := []FileInfo{}

	for _, fileInfo := range fileInfos {
		filePath := fileInfo.absPath
		if !filepath.IsAbs(filePath) {
			logger.Fatalf("Path %v is not absolute path", filePath)
			return
		}
		var file *os.File

		// if file already exists, then pass
		_, found := g.subGeneratorMap[filePath]
		if found {
			continue
		}

		file, err = os.Open(filePath)
		if err != nil {
			logger.Errorf("Could not open file %v", filePath)
			return err
		}
		defer file.Close()

		var stat fs.FileInfo
		stat, err = file.Stat()
		if err != nil {
			return err
		}

		if stat.IsDir() {
			newfiles, err := g.getAllFileFromDir(filePath)
			if err != nil {
				logger.Errorf("getAllFileFromDir error %v", err)
				return err
			}
			files = append(files, newfiles...)
		} else {
			isIdl, _ := g.absPathIsIdl(filePath)
			if !isIdl {
				logger.Infof("file %v is not valid idl", filePath)
				continue
			}
			files = append(files, fileInfo)
		}

	}

	g.filesStack = append(g.filesStack, files...)

	for _, file := range files {
		path := file.absPath
		outputDir, filename := filepath.Split(file.outputPath)

		if g.conf.Task == TASK_FILE_PROTO2THRIFT {
			var generator SubGenerator
			conf := &thriftGeneratorConfig{
				taskType:       g.conf.Task,
				filePath:       path,
				fileName:       filename,
				outputDir:      outputDir,
				useSpaceIndent: g.conf.UseSpaceIndent,
				indentSpace:    g.conf.IndentSpace,
				fieldCase:      g.conf.FieldCase,
				nameCase:       g.conf.NameCase,
				syntax:         g.conf.Syntax,
			}
			generator, err = NewThriftGenerator(conf)
			if err != nil {
				logger.Fatalf("Initializing thrift generator error %v", err)
				return
			}
			g.subGeneratorMap[path] = generator
		} else if g.conf.Task == TASK_FILE_THRIFT2PROTO {
			var generator SubGenerator
			conf := &protoGeneratorConfig{
				taskType:       g.conf.Task,
				filePath:       path,
				fileName:       filename,
				outputDir:      outputDir,
				useSpaceIndent: g.conf.UseSpaceIndent,
				indentSpace:    g.conf.IndentSpace,
				fieldCase:      g.conf.FieldCase,
				nameCase:       g.conf.NameCase,
				syntax:         g.conf.Syntax,
			}
			generator, err = NewProtoGenerator(conf)
			if err != nil {
				logger.Fatalf("Initializing proto generator error %v", err)
				return
			}
			g.subGeneratorMap[path] = generator
		}
	}
	return
}

func (g *generator) initSubGeneratorForRawContent() (err error) {
	logger.Info("initSubGeneratorForRawContent start")

	path := "raw_content"
	g.filesStack = append(g.filesStack, FileInfo{
		absPath: path,
	})
	if g.conf.Task == TASK_CONTENT_PROTO2THRIFT {
		var generator SubGenerator
		conf := &thriftGeneratorConfig{
			taskType:       g.conf.Task,
			rawContent:     g.conf.RawContent,
			filePath:       path,
			useSpaceIndent: g.conf.UseSpaceIndent,
			indentSpace:    g.conf.IndentSpace,
			fieldCase:      g.conf.FieldCase,
			nameCase:       g.conf.NameCase,
			syntax:         g.conf.Syntax,
		}
		generator, err = NewThriftGenerator(conf)
		if err != nil {
			return
		}
		g.subGeneratorMap[path] = generator
	} else if g.conf.Task == TASK_CONTENT_THRIFT2PROTO {
		var generator SubGenerator
		conf := &protoGeneratorConfig{
			taskType:       g.conf.Task,
			rawContent:     g.conf.RawContent,
			filePath:       path,
			useSpaceIndent: g.conf.UseSpaceIndent,
			indentSpace:    g.conf.IndentSpace,
			fieldCase:      g.conf.FieldCase,
			nameCase:       g.conf.NameCase,
			syntax:         g.conf.Syntax,
		}
		generator, err = NewProtoGenerator(conf)
		if err != nil {
			return
		}
		g.subGeneratorMap[path] = generator
	}
	return
}

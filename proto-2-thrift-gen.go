package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/emicklei/proto"
	"github.com/protobuf-thrift/utils"
	"github.com/protobuf-thrift/utils/logger"
	goThrift "github.com/samuel/go-thrift/parser"
)

type thriftGenerator struct {
	conf          *thriftGeneratorConfig
	def           *proto.Proto
	file          *os.File
	thriftContent bytes.Buffer
	thriftAST     *goThrift.Thrift
	newFiles      []FileInfo
	syntax        int
}

type thriftGeneratorConfig struct {
	taskType   int
	filePath   string // absolute path for current file
	fileName   string // relative filename including path for file to be generated
	rawContent string
	outputDir  string // absolute path for output dir

	useSpaceIndent bool
	indentSpace    string
	fieldCase      string
	nameCase       string

	// pb config
	syntax int // 2 or 3
}

func NewThriftGenerator(conf *thriftGeneratorConfig) (res SubGenerator, err error) {
	var parser *proto.Parser
	var file *os.File
	var syntax int
	if conf.taskType == TASK_FILE_PROTO2THRIFT {
		file, err = os.Open(conf.filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		parser = proto.NewParser(file)

		// get syntax from file
		file1, err := os.Open(conf.filePath)
		if err != nil {
			return nil, err
		}
		defer file1.Close()
		content, err := io.ReadAll(file1)
		if err != nil {
			return nil, err
		}
		if strings.Contains(string(content), "syntax = \"proto3\"") {
			syntax = 3
		} else {
			syntax = 2
		}
	} else if conf.taskType == TASK_CONTENT_PROTO2THRIFT {
		if strings.Contains(conf.rawContent, "syntax = \"proto3\"") {
			syntax = 3
		} else {
			syntax = 2
		}
		rd := strings.NewReader(conf.rawContent)
		parser = proto.NewParser(rd)
	}

	definition, err := parser.Parse()
	if err != nil {
		return
	}

	res = &thriftGenerator{
		conf:   conf,
		def:    definition,
		file:   file,
		syntax: syntax,
	}
	return
}

func (g *thriftGenerator) FilePath() (res string) {
	if g.conf.taskType == TASK_CONTENT_PROTO2THRIFT {
		res = ""
	} else {
		res = g.conf.filePath
	}
	return
}

// generate thriftAST from proto ast
func (g *thriftGenerator) Parse() (newFiles []FileInfo, err error) {
	g.thriftAST = &goThrift.Thrift{
		Includes: make(map[string]string),
		Enums:    make(map[string]*goThrift.Enum),
		Structs:  make(map[string]*goThrift.Struct),
		Services: make(map[string]*goThrift.Service),
	}

	proto.Walk(
		g.def,
		proto.WithPackage(g.handlePackage),
		proto.WithImport(g.handleImport),
		proto.WithService(g.handleService),
		proto.WithMessage(g.handleMessage),
		proto.WithEnum(g.handleEnum),
	)

	newFiles = g.newFiles
	return
}

// write thrift code from thriftAST to output
func (g *thriftGenerator) Sink() (err error) {
	g.sinkImport()
	g.sinkNamespace()
	g.sinkEnum()
	g.sinkStruct()
	g.sinkService()

	if g.conf.outputDir != "" {
		var file *os.File
		err = os.MkdirAll(g.conf.outputDir, 0755)
		if err != nil {
			logger.Errorf("Error occurred when MkdirAll %v", g.conf.outputDir)
			return
		}
		outputPath := filepath.Join(g.conf.outputDir, g.conf.fileName)
		file, err = os.Create(outputPath)
		if err != nil {
			logger.Errorf("os.Create file %v error %v", outputPath, err)
			return err
		}
		defer file.Close()
		_, err = file.WriteString(g.thriftContent.String())
	} else {
		f := bufio.NewWriter(os.Stdout)
		defer f.Flush()
		_, err = f.Write(g.thriftContent.Bytes())
	}

	return
}

func (g *thriftGenerator) handlePackage(p *proto.Package) {
	packageName := p.Name
	namespace := make(map[string]string)
	namespace["*"] = packageName
	g.thriftAST.Namespaces = namespace
	return
}

func (g *thriftGenerator) handleImport(i *proto.Import) {
	if g.conf.taskType != TASK_FILE_PROTO2THRIFT {
		return
	}

	fileName := strings.ReplaceAll(i.Filename, ".proto", ".thrift")
	_, name := filepath.Split(fileName)
	g.thriftAST.Includes[name] = fileName

	var newFile FileInfo
	if filepath.IsAbs(i.Filename) {
		relPath, err := filepath.Rel(g.conf.filePath, i.Filename)
		if err != nil {
			logger.Errorf("filepath.Rel %v %v, err %v", g.conf.filePath, i.Filename, err)
			return
		}
		newFile = FileInfo{
			absPath:    i.Filename,
			outputPath: filepath.Join(g.conf.outputDir, strings.ReplaceAll(relPath, ".proto", ".thrift")),
		}
	} else {
		newFile = FileInfo{
			absPath:    filepath.Join(filepath.Dir(g.conf.filePath), i.Filename),
			outputPath: filepath.Join(g.conf.outputDir, strings.ReplaceAll(i.Filename, ".proto", ".thrift")),
		}
	}
	g.newFiles = append(g.newFiles, newFile)
}

func (g *thriftGenerator) handleService(s *proto.Service) {
	methodMap := make(map[string]*goThrift.Method)
	g.thriftAST.Services[s.Name] = &goThrift.Service{
		Name:    s.Name,
		Methods: methodMap,
	}
	for _, ele := range s.Elements {
		field := ele.(*proto.RPC)
		name := field.Name
		args := []*goThrift.Field{
			// since protobuf rpc method request argument dont have name, we use a default name 'req'
			{
				ID:   1,
				Name: "req",
				Type: &goThrift.Type{
					Name: field.RequestType,
				},
			},
		}
		methodMap[name] = &goThrift.Method{
			Name:      name,
			Arguments: args,
			ReturnType: &goThrift.Type{
				Name: field.ReturnsType,
			},
		}
	}
}

func (g *thriftGenerator) handleEnum(s *proto.Enum) {
	valueMap := make(map[string]*goThrift.EnumValue)
	g.thriftAST.Enums[s.Name] = &goThrift.Enum{
		Name:   s.Name,
		Values: valueMap,
	}

	for _, ele := range s.Elements {
		field := ele.(*proto.EnumField)
		name := field.Name
		valueMap[name] = &goThrift.EnumValue{
			Name:  name,
			Value: field.Integer,
		}
	}
}

func (g *thriftGenerator) handleMessage(m *proto.Message) {
	fields := []*goThrift.Field{}
	g.thriftAST.Structs[m.Name] = &goThrift.Struct{
		Name:   m.Name,
		Fields: fields,
	}

	for _, ele := range m.Elements {
		var field *goThrift.Field

		// handle fields except for map
		mes, ok := ele.(*proto.NormalField)
		if ok {
			optional := g.syntax == 2 && mes.Optional
			field = &goThrift.Field{
				ID:       mes.Sequence,
				Name:     mes.Name,
				Optional: optional,
			}

			if mes.Repeated {
				t, err := g.typeConverter(mes.Type)
				if err != nil {
					logger.Error(err)
					continue
				}
				field.Type = &goThrift.Type{
					Name:      "list",
					ValueType: t,
				}
			} else {
				t, err := g.typeConverter(mes.Type)
				if err != nil {
					logger.Error(err)
					continue
				}
				field.Type = t
			}
		} else {
			mes, ok := ele.(*proto.MapField)
			if ok {
				field = &goThrift.Field{
					ID:   mes.Sequence,
					Name: mes.Name,
				}
				keyType, err := g.basicTypeConverter(mes.KeyType)
				if err != nil {
					logger.Errorf("Invalid map key type: %v", mes.KeyType)
				}
				valueType, err := g.typeConverter(mes.Type)
				if err != nil {
					logger.Errorf("Invalid map value type: %v", mes.Type)
					return
				}

				field.Type = &goThrift.Type{
					Name:      "map",
					KeyType:   keyType,
					ValueType: valueType,
				}

			} else {
				logger.Errorf("Unknown invalid proto message field: %+v", mes)
				continue
			}
		}

		// finally append thrift field to fields slice
		fields = append(fields, field)
	}

	g.thriftAST.Structs[m.Name].Fields = fields
}

func (g *thriftGenerator) typeConverter(t string) (res *goThrift.Type, err error) {
	res, err = g.basicTypeConverter(t)
	if err != nil {
		// if t is not a basic type, then we should convert its case, same as name
		res = &goThrift.Type{
			Name: utils.CaseConvert(g.conf.nameCase, t),
		}
		return res, nil
	}
	return
}

func (g *thriftGenerator) basicTypeConverter(t string) (res *goThrift.Type, err error) {
	switch t {
	case "string":
		res = &goThrift.Type{
			Name: "string",
		}
	case "int64":
		res = &goThrift.Type{
			Name: "i64",
		}
	case "int32":
		res = &goThrift.Type{
			Name: "i32",
		}
	case "float", "double":
		res = &goThrift.Type{
			Name: "double",
		}
	case "bool":
		res = &goThrift.Type{
			Name: "bool",
		}
	case "bytes":
		res = &goThrift.Type{
			Name: "binary",
		}
	default:
		err = fmt.Errorf("Invalid basic type %s", t)
	}
	return
}

func (g *thriftGenerator) sinkService() {
	for _, s := range g.thriftAST.Services {
		name := utils.CaseConvert(g.conf.nameCase, s.Name)
		g.thriftContent.WriteString(fmt.Sprintf("\nservice %s {\n", name))
		for _, m := range s.Methods {
			name := utils.CaseConvert(g.conf.nameCase, m.Name)
			g.writeIndent()
			g.thriftContent.WriteString(
				fmt.Sprintf(
					"%s %s (%d: %s %s)\n",
					m.ReturnType.String(),
					name,
					m.Arguments[0].ID,
					m.Arguments[0].Type.String(),
					utils.CaseConvert(g.conf.nameCase, m.Arguments[0].Name),
				),
			)
		}
		g.thriftContent.WriteString("}\n")
	}
}

func (g *thriftGenerator) sinkImport() {
	for _, filePath := range g.thriftAST.Includes {
		g.thriftContent.WriteString(fmt.Sprintf("include \"%s\";\n", filePath))
	}
}

func (g *thriftGenerator) sinkNamespace() {
	for key, name := range g.thriftAST.Namespaces {
		g.thriftContent.WriteString(fmt.Sprintf("namespace %s %s;\n\n", key, name))
	}
}

func (g *thriftGenerator) sinkEnum() {
	for _, enum := range g.thriftAST.Enums {
		name := utils.CaseConvert(g.conf.nameCase, enum.Name)
		g.thriftContent.WriteString(fmt.Sprintf("enum %s {\n", name))
		// since for-range map is random-ordered, we need to sort first, then write
		valueSlice := []*goThrift.EnumValue{}
		for _, value := range enum.Values {
			valueSlice = append(valueSlice, value)
		}
		sort.Slice(valueSlice, func(i, j int) bool {
			return valueSlice[i].Value < valueSlice[j].Value
		})

		for _, field := range valueSlice {
			fieldName := utils.CaseConvert(g.conf.fieldCase, field.Name)
			g.writeIndent()
			g.thriftContent.WriteString(fmt.Sprintf("%s = %d\n", fieldName, field.Value))
		}
		g.thriftContent.WriteString("}\n")
	}
}

func (g *thriftGenerator) sinkStruct() {
	for _, sct := range g.thriftAST.Structs {
		name := utils.CaseConvert(g.conf.nameCase, sct.Name)
		g.thriftContent.WriteString(fmt.Sprintf("struct %s {\n", name))

		for _, field := range sct.Fields {
			typeName := field.Type.String()
			fieldName := utils.CaseConvert(g.conf.fieldCase, field.Name)
			g.writeIndent()
			optStr := ""
			if field.Optional {
				optStr = " optional"
			}
			g.thriftContent.WriteString(fmt.Sprintf("%d:%s %s %s\n", field.ID, optStr, typeName, fieldName))
		}

		g.thriftContent.WriteString("}\n")
	}
}

func (g *thriftGenerator) writeIndent() {
	if g.conf.useSpaceIndent {
		spaceCount, _ := strconv.Atoi(g.conf.indentSpace)
		for i := 0; i < spaceCount; i++ {
			g.thriftContent.WriteString(" ")
		}
	} else {
		g.thriftContent.WriteString("	")
	}
}

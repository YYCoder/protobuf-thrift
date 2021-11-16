package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/YYCoder/protobuf-thrift/utils"
	"github.com/YYCoder/protobuf-thrift/utils/logger"
	"github.com/YYCoder/thrifter"
	"github.com/emicklei/proto"
)

type protoGenerator struct {
	conf         *protoGeneratorConfig
	def          *thrifter.Thrift
	file         *os.File
	protoContent bytes.Buffer
	// protoAST     *proto.Proto
}

type protoGeneratorConfig struct {
	taskType   int
	filePath   string // absolute path for current file
	fileName   string // output file name, including extension
	rawContent string
	outputDir  string // absolute path for output dir

	useSpaceIndent bool
	indentSpace    string
	fieldCase      string
	nameCase       string

	// pb config
	syntax int // 2 or 3
}

func NewProtoGenerator(conf *protoGeneratorConfig) (res SubGenerator, err error) {
	var parser *thrifter.Parser
	var file *os.File
	var definition *thrifter.Thrift
	if conf.taskType == TASK_FILE_THRIFT2PROTO {
		file, err = os.Open(conf.filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		parser = thrifter.NewParser(file, false)
		definition, err = parser.Parse(file.Name())

	} else if conf.taskType == TASK_CONTENT_THRIFT2PROTO {
		rd := strings.NewReader(conf.rawContent)
		parser = thrifter.NewParser(rd, false)
		definition, err = parser.Parse("INPUT")
	}

	if err != nil {
		return
	}

	res = &protoGenerator{
		conf: conf,
		def:  definition,
		file: file,
	}
	return
}

func (g *protoGenerator) FilePath() (res string) {
	if g.conf.taskType == TASK_CONTENT_THRIFT2PROTO {
		res = ""
	} else {
		res = g.conf.filePath
	}
	return
}

// parse thrift ast, return absolute file paths included by current file
func (g *protoGenerator) Parse() (newFiles []FileInfo, err error) {
	g.protoAST = &proto.Proto{}

	g.handleSyntax()
	g.handleNamespace(g.def.Namespaces)
	if g.conf.taskType == TASK_FILE_THRIFT2PROTO {
		for k, i := range g.def.Includes {
			newFiles = append(newFiles, g.handleIncludes(k, i))
		}
	}
	for _, i := range g.def.Enums {
		g.handleEnum(i)
	}
	for _, i := range g.def.Structs {
		g.handleStruct(i)
	}
	for _, i := range g.def.Services {
		g.handleService(i)
	}
	return
}

func (g *protoGenerator) handleSyntax() {
	protoSyntax := &proto.Syntax{
		Value:  fmt.Sprintf("proto%d", g.conf.syntax),
		Parent: g.protoAST,
	}
	g.protoAST.Elements = append(g.protoAST.Elements, protoSyntax)
	return
}

func (g *protoGenerator) handleNamespace(n map[string]string) {
	var packageName string
	for _, name := range n {
		packageName = name
	}
	protoPackage := &proto.Package{
		Name:   packageName,
		Parent: g.protoAST,
	}
	g.protoAST.Elements = append(g.protoAST.Elements, protoPackage)
	return
}

func (g *protoGenerator) handleIncludes(name string, path string) (newFile FileInfo) {
	if filepath.IsAbs(path) {
		relPath, err := filepath.Rel(g.conf.filePath, path)
		if err != nil {
			logger.Errorf("filepath.Rel %v %v, err %v", g.conf.filePath, path, err)
			return
		}
		newFile = FileInfo{
			absPath:    path,
			outputPath: filepath.Join(g.conf.outputDir, strings.ReplaceAll(relPath, ".thrift", ".proto")),
		}
	} else {
		newFile = FileInfo{
			absPath:    filepath.Join(filepath.Dir(g.conf.filePath), path),
			outputPath: filepath.Join(g.conf.outputDir, strings.ReplaceAll(path, ".thrift", ".proto")),
		}
	}

	protoImport := &proto.Import{
		Filename: strings.ReplaceAll(path, ".thrift", ".proto"),
		Parent:   g.protoAST,
	}
	g.protoAST.Elements = append(g.protoAST.Elements, protoImport)
	return
}

func (g *protoGenerator) handleService(s *goThrift.Service) {
	protoService := &proto.Service{
		Name:   s.Name,
		Parent: g.protoAST,
	}
	for _, i := range s.Methods {
		// type convert
		var reqType, resType string
		var err error
		if len(i.Arguments) > 0 {
			reqType, err = g.typeConverter(i.Arguments[0].Type)
			if err != nil {
				logger.Errorf("Invalid requestType %v", err)
				continue
			}
		}
		if i.ReturnType != nil {
			resType, err = g.typeConverter(i.ReturnType)
			if err != nil {
				logger.Errorf("Invalid returnType %v", err)
				continue
			}
		}

		method := &proto.RPC{
			Name:        i.Name,
			RequestType: reqType,
			ReturnsType: resType,
			Parent:      protoService,
		}

		if len(i.Annotations) > 0 {
			// add options
			o := &proto.Option{
				// if use http option, default name is (google.api.http)
				Name:   "(google.api.http)",
				Parent: method,
				Constant: proto.Literal{
					OrderedMap: []*proto.NamedLiteral{},
				},
			}
			options := []proto.Visitee{o}
			for _, a := range i.Annotations {
				lit := &proto.Literal{
					Source:    a.Value,
					QuoteRune: 34,
					IsString:  true,
				}
				o.Constant.OrderedMap = append(o.Constant.OrderedMap, &proto.NamedLiteral{
					Literal:     lit,
					Name:        a.Name,
					PrintsColon: true,
				})
			}
			method.Elements = options
		}

		// finally append method to service
		protoService.Elements = append(protoService.Elements, method)
	}
	g.protoAST.Elements = append(g.protoAST.Elements, protoService)
	return
}

func (g *protoGenerator) handleEnum(e *goThrift.Enum) {
	protoEnum := &proto.Enum{
		Name:   e.Name,
		Parent: g.protoAST,
	}

	// since for-range map is random-ordered, we need to sort first
	hasUnknownField := false
	valueSlice := []proto.EnumField{}
	for _, i := range e.Values {
		if i.Value == 0 {
			hasUnknownField = true
		}
		valueSlice = append(valueSlice, proto.EnumField{
			Name:    i.Name,
			Integer: i.Value,
		})
	}
	// for protobuf which syntax is 3, enum field must have an default field which value is 0
	if g.conf.syntax == 3 && !hasUnknownField {
		valueSlice = append(valueSlice, proto.EnumField{
			Name:    fmt.Sprintf("%v_Unknown", e.Name),
			Integer: 0,
		})
	}
	sort.Slice(valueSlice, func(i, j int) bool {
		return valueSlice[i].Integer < valueSlice[j].Integer
	})

	visiteeSlice := []proto.Visitee{}
	for _, v := range valueSlice {
		visiteeSlice = append(visiteeSlice, &proto.EnumField{
			Name:    v.Name,
			Integer: v.Integer,
		})
	}

	protoEnum.Elements = visiteeSlice
	g.protoAST.Elements = append(g.protoAST.Elements, protoEnum)
	return
}

func (g *protoGenerator) handleStruct(s *goThrift.Struct) {
	message := &proto.Message{
		Name:   s.Name,
		Parent: g.protoAST,
	}

	elements := []proto.Visitee{}
	for _, f := range s.Fields {
		switch f.Type.Name {
		case "list":
			fieldType, _ := g.typeConverter(f.Type.ValueType)
			pbField := &proto.Field{
				Name:     f.Name,
				Parent:   message,
				Sequence: f.ID,
				Type:     fieldType,
			}
			pbNormalField := &proto.NormalField{
				Field:    pbField,
				Repeated: true,
			}
			elements = append(elements, pbNormalField)
		case "map":
			fieldType, _ := g.typeConverter(f.Type.ValueType)
			keyType, _ := g.basicTypeConverter(f.Type.KeyType)
			pbField := &proto.Field{
				Name:     f.Name,
				Parent:   message,
				Sequence: f.ID,
				Type:     fieldType,
			}
			pbMapField := &proto.MapField{
				Field:   pbField,
				KeyType: keyType,
			}

			elements = append(elements, pbMapField)
		// since proto doesn't have type set, we don't need to parse this type
		case "set":
			logger.Warnf("Protobuf doesn't have type set")
			continue
		default:
			optional := g.conf.syntax == 2 && f.Optional

			fieldType, _ := g.typeConverter(f.Type)
			pbField := &proto.Field{
				Name:     f.Name,
				Parent:   message,
				Sequence: f.ID,
				Type:     fieldType,
			}
			pbNormalField := &proto.NormalField{
				Field:    pbField,
				Optional: optional,
			}
			elements = append(elements, pbNormalField)
		}
	}
	message.Elements = elements

	g.protoAST.Elements = append(g.protoAST.Elements, message)
	return
}

func (g *protoGenerator) typeConverter(t *goThrift.Type) (res string, err error) {
	res, err = g.basicTypeConverter(t)
	if err != nil {
		// if t is not a basic type, then we should convert its case, same as name
		res = utils.CaseConvert(g.conf.nameCase, t.Name)
		return res, nil
	}
	return
}

func (g *protoGenerator) basicTypeConverter(t *goThrift.Type) (res string, err error) {
	switch t.Name {
	case "string":
		res = "string"
	case "i64":
		res = "int64"
	case "i32":
		res = "int32"
	case "double":
		res = "double"
	case "bool":
		res = "bool"
	case "binary":
		res = "bytes"
	default:
		err = fmt.Errorf("Invalid basic type %s", t)
	}
	return
}

// write thrift code from thriftAST to output
func (g *protoGenerator) Sink() (err error) {
	// start traverse
	g.protoAST.Accept(g)

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
		_, err = file.WriteString(g.protoContent.String())
	} else {
		f := bufio.NewWriter(os.Stdout)
		defer f.Flush()
		_, err = f.Write(g.protoContent.Bytes())
	}

	return
}

func (g *protoGenerator) VisitMessage(item *proto.Message) {
	name := utils.CaseConvert(g.conf.nameCase, item.Name)
	g.protoContent.WriteString(fmt.Sprintf("message %s {\n", name))
	for _, e := range item.Elements {
		e.Accept(g)
	}
	g.protoContent.WriteString("}\n")
	return
}
func (g *protoGenerator) VisitService(item *proto.Service) {
	name := utils.CaseConvert(g.conf.nameCase, item.Name)
	g.protoContent.WriteString(fmt.Sprintf("service %s {\n", name))
	for _, e := range item.Elements {
		e.Accept(g)
	}
	g.protoContent.WriteString("}\n")
	return
}
func (g *protoGenerator) VisitSyntax(item *proto.Syntax) {
	g.protoContent.WriteString(fmt.Sprintf("syntax = \"%s\";\n", item.Value))
	return
}
func (g *protoGenerator) VisitPackage(item *proto.Package) {
	g.protoContent.WriteString(fmt.Sprintf("package %s;\n\n", item.Name))
	return
}
func (g *protoGenerator) VisitOption(item *proto.Option) {
	if item.Name == "(google.api.http)" {
		g.writeIndent()
		g.writeIndent()
		g.protoContent.WriteString(fmt.Sprintf("option %s = {\n", item.Name))
		for _, e := range item.Constant.OrderedMap {
			g.writeIndent()
			g.writeIndent()
			g.writeIndent()
			g.protoContent.WriteString(fmt.Sprintf("%s: \"%s\"\n", e.Name, e.Source))
		}
		g.writeIndent()
		g.writeIndent()
		g.protoContent.WriteString("};\n")
	}
	return
}
func (g *protoGenerator) VisitImport(item *proto.Import) {
	g.protoContent.WriteString(fmt.Sprintf("import \"%s\";\n", item.Filename))
	return
}
func (g *protoGenerator) VisitNormalField(item *proto.NormalField) {
	name := utils.CaseConvert(g.conf.fieldCase, item.Name)
	g.writeIndent()
	if item.Optional {
		g.protoContent.WriteString("optional ")
	}
	if item.Repeated {
		g.protoContent.WriteString(fmt.Sprintf("repeated %s %s = %d;\n", item.Type, name, item.Sequence))
	} else {
		g.protoContent.WriteString(fmt.Sprintf("%s %s = %d;\n", item.Type, name, item.Sequence))
	}
	return
}
func (g *protoGenerator) VisitEnumField(item *proto.EnumField) {
	fieldName := utils.CaseConvert(g.conf.fieldCase, item.Name)
	g.writeIndent()
	g.protoContent.WriteString(fmt.Sprintf("%s = %d;\n", fieldName, item.Integer))
	return
}
func (g *protoGenerator) VisitEnum(item *proto.Enum) {
	name := utils.CaseConvert(g.conf.nameCase, item.Name)
	g.protoContent.WriteString(fmt.Sprintf("enum %s {\n", name))
	for _, ele := range item.Elements {
		ele.Accept(g)
	}
	g.protoContent.WriteString("}\n")
	return
}
func (g *protoGenerator) VisitComment(item *proto.Comment) {
	return
}
func (g *protoGenerator) VisitOneof(item *proto.Oneof) {
	return
}
func (g *protoGenerator) VisitOneofField(item *proto.OneOfField) {
	return
}
func (g *protoGenerator) VisitReserved(item *proto.Reserved) {
	return
}
func (g *protoGenerator) VisitRPC(item *proto.RPC) {
	name := utils.CaseConvert(g.conf.nameCase, item.Name)
	reqName := utils.CaseConvert(g.conf.nameCase, item.RequestType)
	resName := utils.CaseConvert(g.conf.nameCase, item.ReturnsType)
	if len(item.Elements) > 0 {
		g.writeIndent()
		g.protoContent.WriteString(fmt.Sprintf("rpc %s(%s) returns (%s) {\n", name, reqName, resName))
		for _, e := range item.Elements {
			e.Accept(g)
		}
		g.writeIndent()
		g.protoContent.WriteString("}\n")
	} else {
		g.writeIndent()
		g.protoContent.WriteString(fmt.Sprintf("rpc %s(%s) returns (%s) {}\n", name, reqName, resName))
	}
	return
}
func (g *protoGenerator) VisitMapField(item *proto.MapField) {
	name := utils.CaseConvert(g.conf.fieldCase, item.Name)
	g.writeIndent()
	g.protoContent.WriteString(fmt.Sprintf("map<%s, %s> %s = %d;\n", item.KeyType, item.Type, name, item.Sequence))
	return
}

// proto 2
func (g *protoGenerator) VisitGroup(item *proto.Group) {
	return
}
func (g *protoGenerator) VisitExtensions(item *proto.Extensions) {
	return
}

func (g *protoGenerator) writeIndent() {
	if g.conf.useSpaceIndent {
		spaceCount, _ := strconv.Atoi(g.conf.indentSpace)
		for i := 0; i < spaceCount; i++ {
			g.protoContent.WriteString(" ")
		}
	} else {
		g.protoContent.WriteString("	")
	}
	return
}

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

	"github.com/YYCoder/protobuf-thrift/utils"
	"github.com/YYCoder/protobuf-thrift/utils/logger"
	"github.com/YYCoder/thrifter"
	"github.com/emicklei/proto"
)

type thriftGenerator struct {
	conf          *thriftGeneratorConfig
	def           *proto.Proto
	file          *os.File
	thriftContent bytes.Buffer
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

// Iterate over each declare and convert it to thrift declaration.
func (g *thriftGenerator) Parse() (newFiles []FileInfo, err error) {
	for _, e := range g.def.Elements {
		switch e.(type) {
		case *proto.Package:
			ele := e.(*proto.Package)
			g.handleComment(ele.Comment, false, 0)
			g.handlePackage(ele)
		case *proto.Import:
			ele := e.(*proto.Import)
			g.handleComment(ele.Comment, false, 0)
			g.handleImport(ele)
		case *proto.Service:
			ele := e.(*proto.Service)
			g.handleComment(ele.Comment, false, 0)
			g.handleService(ele)
		case *proto.Message:
			ele := e.(*proto.Message)
			g.handleComment(ele.Comment, false, 0)
			g.handleMessage(ele)
		case *proto.Enum:
			ele := e.(*proto.Enum)
			g.handleComment(ele.Comment, false, 0)
			g.handleEnum(ele)
		// syntaxes that thrift does not support, only handle comments.
		case *proto.Extensions:
		case *proto.Syntax:
		case *proto.Option:
		case *proto.Comment:
			ele := e.(*proto.Comment)
			g.handleComment(ele, false, 0)
		default:
			// logger.Infof("other: %+v\n", e)
		}
	}

	newFiles = g.newFiles
	return
}

// Write thrift code from thriftContent to output.
func (g *thriftGenerator) Sink() (err error) {
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

func (g *thriftGenerator) handleComment(ele *proto.Comment, inline bool, indentCount int) (err error) {
	// since we only want to read comments, let's just assert it to a random type.
	if ele == nil {
		return
	}
	if inline {
		for i := 0; i < indentCount; i++ {
			g.writeIndent()
		}
		if ele.Cstyle {
			// concat multiple line comment into one line.
			g.thriftContent.WriteString("/*")
			for _, line := range ele.Lines {
				g.thriftContent.WriteString(fmt.Sprintf("%s ", line))
			}
			g.thriftContent.WriteString("*/")
		} else {
			if ele.ExtraSlash {
				g.thriftContent.WriteString("/")
			}
			g.thriftContent.WriteString(fmt.Sprintf("//%s", ele.Lines[0]))
		}
	} else {
		if ele.Cstyle {
			for i := 0; i < indentCount; i++ {
				g.writeIndent()
			}
			g.thriftContent.WriteString("/**")
			for _, comment := range ele.Lines {
				g.thriftContent.WriteString("\n")
				for i := 0; i < indentCount; i++ {
					g.writeIndent()
				}
				g.thriftContent.WriteString(" *")
				g.thriftContent.WriteString(comment)
			}
			g.thriftContent.WriteString("\n")
			for i := 0; i < indentCount; i++ {
				g.writeIndent()
			}
			g.thriftContent.WriteString(" */\n")
		} else {
			for _, line := range ele.Lines {
				for i := 0; i < indentCount; i++ {
					g.writeIndent()
				}
				g.thriftContent.WriteString("//")
				if ele.ExtraSlash {
					g.thriftContent.WriteString("/")
				}
				g.thriftContent.WriteString(line)
				g.thriftContent.WriteString("\n")
			}
		}
	}
	return
}

func (g *thriftGenerator) handlePackage(p *proto.Package) {
	g.thriftContent.WriteString(fmt.Sprintf("namespace * %s;\n\n", p.Name))
	return
}

// Analyze proto import declaration and append it to newFiles in order to recursively parse imported files. Then, convert import declaration to thrift include declaration.
func (g *thriftGenerator) handleImport(i *proto.Import) {
	if g.conf.taskType != TASK_FILE_PROTO2THRIFT {
		return
	}

	fileName := strings.ReplaceAll(i.Filename, ".proto", ".thrift")
	// analyze dependency
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

	// convert import declaration
	// ! NOTE: thrift include can not using semicolon as end of declaration.
	g.thriftContent.WriteString(fmt.Sprintf("include \"%s\"\n", fileName))
}

func (g *thriftGenerator) handleService(s *proto.Service) {
	name := utils.CaseConvert(g.conf.nameCase, s.Name)
	g.thriftContent.WriteString(fmt.Sprintf("\nservice %s {\n", name))
	for _, m := range s.Elements {
		// if element is a comment
		comment, ok := m.(*proto.Comment)
		if ok {
			g.handleComment(comment, false, 1)
			continue
		}

		field := m.(*proto.RPC)
		// handle comment first, because proto can only parse comment above rpc declaration.
		if field.Comment != nil {
			g.handleComment(field.Comment, false, 1)
		}
		name := utils.CaseConvert(g.conf.nameCase, field.Name)
		g.writeIndent()
		g.thriftContent.WriteString(
			fmt.Sprintf(
				"%s %s (%d: %s %s)\n",
				field.ReturnsType,
				name,
				1,
				field.RequestType,
				// since protobuf rpc method request argument dont have name, we use a default name 'req'
				utils.CaseConvert(g.conf.nameCase, "req"),
			),
		)
	}
	g.thriftContent.WriteString("}\n")
}

func (g *thriftGenerator) handleEnum(s *proto.Enum) {
	name := utils.CaseConvert(g.conf.nameCase, s.Name)
	g.thriftContent.WriteString(fmt.Sprintf("enum %s {\n", name))
	// since for-range map is random-ordered, we need to sort first, then write
	valueSlice := []*proto.EnumField{}
	for _, value := range s.Elements {
		ele := value.(*proto.EnumField)
		valueSlice = append(valueSlice, ele)
	}
	sort.Slice(valueSlice, func(i, j int) bool {
		return valueSlice[i].Integer < valueSlice[j].Integer
	})

	for _, field := range valueSlice {
		// handle comment above field
		if field.Comment != nil {
			g.handleComment(field.Comment, false, 1)
		}

		fieldName := utils.CaseConvert(g.conf.fieldCase, field.Name)
		g.writeIndent()
		g.thriftContent.WriteString(fmt.Sprintf("%s = %d", fieldName, field.Integer))
		// handle comment after field line
		if field.InlineComment != nil {
			g.thriftContent.WriteString(" ")
			g.handleComment(field.InlineComment, true, 0)
		}
		g.thriftContent.WriteString("\n")
	}
	g.thriftContent.WriteString("}\n")
}

// Handle protobuf message declaration.
// 1. use thrifter ast node to simplify generation of thrift code.
// 2. if it has nested enum or message, will prefix its name with outer message name to identify.
func (g *thriftGenerator) handleMessage(m *proto.Message) {
	name := utils.CaseConvert(g.conf.nameCase, m.Name)
	g.thriftContent.WriteString(fmt.Sprintf("struct %s {\n", name))
	nestedEnums := []*proto.Enum{}
	nestedMessages := []*proto.Message{}

	// in case ident is a nested enum or message name, check first.
	// NOTE: nested fields must declare before other fields refer to them, otherwise it can't be identified.
	var getIdentFieldName = func(ident string) string {
		nestedName := utils.CaseConvert(g.conf.fieldCase, fmt.Sprintf("%s%s", m.Name, ident))
		for _, e := range nestedEnums {
			if e.Name == nestedName {
				return nestedName
			}
		}
		for _, e := range nestedMessages {
			if e.Name == nestedName {
				return nestedName
			}
		}
		return ident
	}

	for _, ele := range m.Elements {
		var field *thrifter.Field
		var inlineComment, comment *proto.Comment

		switch ele.(type) {
		case *proto.NormalField:
			mes := ele.(*proto.NormalField)
			inlineComment = mes.InlineComment
			comment = mes.Comment
			optional := g.syntax == 2 && mes.Optional
			field = &thrifter.Field{
				ID:    mes.Sequence,
				Ident: mes.Name,
			}
			if optional {
				field.Requiredness = "optional"
			}

			if mes.Repeated {
				t, err := g.typeConverter(mes.Type)
				if err != nil {
					logger.Error(err)
					continue
				}
				field.FieldType = &thrifter.FieldType{
					Type: thrifter.FIELD_TYPE_LIST,
					List: &thrifter.ListType{
						Elem: &thrifter.FieldType{
							Type:  thrifter.FIELD_TYPE_IDENT,
							Ident: getIdentFieldName(t),
						},
					},
				}
			} else {
				t, err := g.typeConverter(mes.Type)
				if err != nil {
					logger.Error(err)
					continue
				}
				field.FieldType = &thrifter.FieldType{
					Type:  thrifter.FIELD_TYPE_IDENT,
					Ident: getIdentFieldName(t),
				}
			}

			g.handleField(field, comment, inlineComment)
		case *proto.MapField:
			mes, ok := ele.(*proto.MapField)
			inlineComment = mes.InlineComment
			comment = mes.Comment
			if ok {
				field = &thrifter.Field{
					ID:    mes.Sequence,
					Ident: mes.Name,
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

				field.FieldType = &thrifter.FieldType{
					Type: thrifter.FIELD_TYPE_MAP,
					Map: &thrifter.MapType{
						Key: &thrifter.FieldType{
							Type:  thrifter.FIELD_TYPE_IDENT,
							Ident: getIdentFieldName(keyType),
						},
						Value: &thrifter.FieldType{
							Type:  thrifter.FIELD_TYPE_IDENT,
							Ident: getIdentFieldName(valueType),
						},
					},
				}

			} else {
				logger.Errorf("Unknown invalid proto message field: %+v", mes)
				continue
			}

			g.handleField(field, comment, inlineComment)
		case *proto.Enum:
			mes := ele.(*proto.Enum)
			mes.Name = fmt.Sprintf("%s%s", m.Name, mes.Name)
			nestedEnums = append(nestedEnums, mes)
		case *proto.Message:
			mes := ele.(*proto.Message)
			mes.Name = fmt.Sprintf("%s%s", m.Name, mes.Name)
			nestedMessages = append(nestedMessages, mes)
		}
	}

	// done handling message
	g.thriftContent.WriteString("}\n")

	for _, e := range nestedEnums {
		g.handleEnum(e)
	}

	for _, e := range nestedMessages {
		g.handleMessage(e)
	}
}

// Convert message field to thrift field type by thrifter Field node.
func (g *thriftGenerator) handleField(field *thrifter.Field, comment *proto.Comment, inlineComment *proto.Comment) {
	// convert field type string
	typeStr := ""
	switch field.FieldType.Type {
	case thrifter.FIELD_TYPE_LIST:
		typeStr = fmt.Sprintf("list<%s>", field.FieldType.List.Elem.Ident)
	case thrifter.FIELD_TYPE_MAP:
		typeStr = fmt.Sprintf("map<%s, %s>", field.FieldType.Map.Key.Ident, field.FieldType.Map.Value.Ident)
	case thrifter.FIELD_TYPE_IDENT:
		typeStr = field.FieldType.Ident
	default:
		logger.Errorf("Unknown thrift field type: %+v", field)
		return
	}

	// handle comment above field
	if comment != nil {
		g.handleComment(comment, false, 1)
	}

	fieldName := utils.CaseConvert(g.conf.fieldCase, field.Ident)
	g.writeIndent()
	optStr := ""
	if field.Requiredness == "optional" {
		optStr = " optional"
	}
	g.thriftContent.WriteString(fmt.Sprintf("%d:%s %s %s", field.ID, optStr, typeStr, fieldName))

	// handle comment after field line
	if inlineComment != nil {
		g.thriftContent.WriteString(" ")
		g.handleComment(inlineComment, true, 0)
	}

	g.thriftContent.WriteString("\n")
}

func (g *thriftGenerator) typeConverter(t string) (res string, err error) {
	res, err = g.basicTypeConverter(t)
	if err != nil {
		// if t is not a basic type, then we should convert its case, same as name
		res = utils.CaseConvert(g.conf.nameCase, t)
		return res, nil
	}
	return
}

func (g *thriftGenerator) basicTypeConverter(t string) (res string, err error) {
	switch t {
	case "string":
		res = "string"
	case "int64":
		res = "i64"
	case "int32":
		res = "i32"
	case "float", "double":
		res = "double"
	case "bool":
		res = "bool"
	case "bytes":
		res = "binary"
	default:
		err = fmt.Errorf("Invalid basic type %s", t)
	}
	return
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

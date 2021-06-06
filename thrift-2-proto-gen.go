package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/emicklei/proto"
	"github.com/protobuf-thrift/utils"
	"github.com/protobuf-thrift/utils/logger"
	goThrift "github.com/samuel/go-thrift/parser"
)

type ProtoGenerator struct {
	conf         *RunnerConfig
	def          *goThrift.Thrift
	file         *os.File
	rawContent   string
	protoContent bytes.Buffer
	protoAST     *proto.Proto
}

func NewProtoGenerator(conf *RunnerConfig, filePath string) (res *ProtoGenerator, err error) {
	var parser *goThrift.Parser
	var rawContent string
	var file *os.File
	var definition *goThrift.Thrift
	if conf.Task == TASK_FILE_THRIFT2PROTO {
		file, err = os.Open(filePath)
		if err != nil {
			return nil, err
		}
		var content []byte
		content, err = io.ReadAll(file)
		rawContent = string(content)

		// if os.File is been read by io.ReadAll, then goThrift.NewParser can't read it, so use
		// os.Open twice
		file, err = os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		parser = &goThrift.Parser{Filesystem: nil}
		definition, err = parser.Parse(file)

	} else if conf.Task == TASK_CONTENT_THRIFT2PROTO {
		rd := strings.NewReader(conf.RawContent)

		rawContent = conf.RawContent
		parser = &goThrift.Parser{Filesystem: nil}
		definition, err = parser.Parse(rd)
	}

	if err != nil {
		return
	}

	res = &ProtoGenerator{
		conf:       conf,
		def:        definition,
		file:       file,
		rawContent: rawContent,
	}
	return
}

func (g *ProtoGenerator) Generate() (err error) {
	if err = g.parse(); err != nil {
		return
	}
	if err = g.sink(); err != nil {
		return
	}
	return
}

// generate protoAST from thrift ast
func (g *ProtoGenerator) parse() (err error) {
	g.protoAST = &proto.Proto{}
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

func (g *ProtoGenerator) handleService(s *goThrift.Service) {
	protoService := &proto.Service{
		Name:   s.Name,
		Parent: g.protoAST,
	}
	for _, i := range s.Methods {
		// type convert
		reqType, err := g.typeConverter(i.Arguments[0].Type)
		if err != nil {
			logger.Errorf("Invalid requestType %v", err)
			continue
		}
		resType, err := g.typeConverter(i.ReturnType)
		if err != nil {
			logger.Errorf("Invalid returnType %v", err)
			continue
		}

		method := &proto.RPC{
			Name:        i.Name,
			RequestType: reqType,
			ReturnsType: resType,
			Parent:      protoService,
		}
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

		// finally append method to service
		protoService.Elements = append(protoService.Elements, method)
	}
	g.protoAST.Elements = append(g.protoAST.Elements, protoService)
	return
}

func (g *ProtoGenerator) handleEnum(e *goThrift.Enum) {
	protoEnum := &proto.Enum{
		Name:   e.Name,
		Parent: g.protoAST,
	}

	// since for-range map is random-ordered, we need to sort first
	valueSlice := []proto.EnumField{}
	for _, i := range e.Values {
		valueSlice = append(valueSlice, proto.EnumField{
			Name:    i.Name,
			Integer: i.Value,
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

func (g *ProtoGenerator) handleStruct(s *goThrift.Struct) {
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
			fieldType, _ := g.typeConverter(f.Type)
			pbField := &proto.Field{
				Name:     f.Name,
				Parent:   message,
				Sequence: f.ID,
				Type:     fieldType,
			}
			pbNormalField := &proto.NormalField{
				Field: pbField,
			}
			elements = append(elements, pbNormalField)
		}
	}
	message.Elements = elements

	g.protoAST.Elements = append(g.protoAST.Elements, message)
	return
}

func (g *ProtoGenerator) typeConverter(t *goThrift.Type) (res string, err error) {
	res, err = g.basicTypeConverter(t)
	if err != nil {
		// if t is not a basic type, then we should convert its case, same as name
		res = utils.CaseConvert(g.conf.NameCase, t.Name)
		return res, nil
	}
	return
}

func (g *ProtoGenerator) basicTypeConverter(t *goThrift.Type) (res string, err error) {
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
func (g *ProtoGenerator) sink() (err error) {
	// start traverse
	g.protoAST.Accept(g)

	if g.conf.OutputPath != "" {
		var file *os.File
		file, err = os.Create(g.conf.OutputPath)
		if err != nil {
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

func (g *ProtoGenerator) VisitMessage(item *proto.Message) {
	name := utils.CaseConvert(g.conf.NameCase, item.Name)
	g.protoContent.WriteString(fmt.Sprintf("message %s {\n", name))
	for _, e := range item.Elements {
		e.Accept(g)
	}
	g.protoContent.WriteString("}\n")
	return
}
func (g *ProtoGenerator) VisitService(item *proto.Service) {
	name := utils.CaseConvert(g.conf.NameCase, item.Name)
	g.protoContent.WriteString(fmt.Sprintf("service %s {\n", name))
	for _, e := range item.Elements {
		e.Accept(g)
	}
	g.protoContent.WriteString("}\n")
	return
}
func (g *ProtoGenerator) VisitSyntax(item *proto.Syntax) {
	return
}
func (g *ProtoGenerator) VisitPackage(item *proto.Package) {
	return
}
func (g *ProtoGenerator) VisitOption(item *proto.Option) {
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
func (g *ProtoGenerator) VisitImport(item *proto.Import) {
	return
}
func (g *ProtoGenerator) VisitNormalField(item *proto.NormalField) {
	name := utils.CaseConvert(g.conf.FieldCase, item.Name)
	g.writeIndent()
	if item.Repeated {
		g.protoContent.WriteString(fmt.Sprintf("repeated %s %s = %d;\n", item.Type, name, item.Sequence))
	} else {
		g.protoContent.WriteString(fmt.Sprintf("%s %s = %d;\n", item.Type, name, item.Sequence))
	}
	return
}
func (g *ProtoGenerator) VisitEnumField(item *proto.EnumField) {
	fieldName := utils.CaseConvert(g.conf.FieldCase, item.Name)
	g.writeIndent()
	g.protoContent.WriteString(fmt.Sprintf("%s = %d;\n", fieldName, item.Integer))
	return
}
func (g *ProtoGenerator) VisitEnum(item *proto.Enum) {
	name := utils.CaseConvert(g.conf.NameCase, item.Name)
	g.protoContent.WriteString(fmt.Sprintf("enum %s {\n", name))
	for _, ele := range item.Elements {
		ele.Accept(g)
	}
	g.protoContent.WriteString("}\n")
	return
}
func (g *ProtoGenerator) VisitComment(item *proto.Comment) {
	return
}
func (g *ProtoGenerator) VisitOneof(item *proto.Oneof) {
	return
}
func (g *ProtoGenerator) VisitOneofField(item *proto.OneOfField) {
	return
}
func (g *ProtoGenerator) VisitReserved(item *proto.Reserved) {
	return
}
func (g *ProtoGenerator) VisitRPC(item *proto.RPC) {
	name := utils.CaseConvert(g.conf.NameCase, item.Name)
	reqName := utils.CaseConvert(g.conf.NameCase, item.RequestType)
	resName := utils.CaseConvert(g.conf.NameCase, item.ReturnsType)
	g.writeIndent()
	g.protoContent.WriteString(fmt.Sprintf("rpc %s(%s) returns (%s) {\n", name, reqName, resName))
	for _, e := range item.Elements {
		e.Accept(g)
	}
	g.writeIndent()
	g.protoContent.WriteString("}\n")
	return
}
func (g *ProtoGenerator) VisitMapField(item *proto.MapField) {
	name := utils.CaseConvert(g.conf.FieldCase, item.Name)
	g.writeIndent()
	g.protoContent.WriteString(fmt.Sprintf("map<%s, %s> %s = %d;\n", item.KeyType, item.Type, name, item.Sequence))
	return
}

// proto 2
func (g *ProtoGenerator) VisitGroup(item *proto.Group) {
	return
}
func (g *ProtoGenerator) VisitExtensions(item *proto.Extensions) {
	return
}

func (g *ProtoGenerator) writeIndent() {
	if g.conf.UseSpaceIndent {
		spaceCount, _ := strconv.Atoi(g.conf.IndentSpace)
		for i := 0; i < spaceCount; i++ {
			g.protoContent.WriteString(" ")
		}
	} else {
		g.protoContent.WriteString("	")
	}
	return
}

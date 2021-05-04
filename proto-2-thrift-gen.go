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
	"github.com/protobuf-thrift/ast/thrift"
	"github.com/protobuf-thrift/utils"
	"github.com/protobuf-thrift/utils/logger"
)

type ThriftGenerator struct {
	conf          *RunnerConfig
	def           *proto.Proto
	file          *os.File
	rawContent    string
	thriftContent bytes.Buffer
	thriftAST     *thrift.Thrift
}

func NewThriftGenerator(conf *RunnerConfig, filePath string) (res *ThriftGenerator, err error) {
	var parser *proto.Parser
	var rawContent string
	var file *os.File
	if conf.Task == TASK_FILE_PROTO2THRIFT {
		file, err = os.Open(filePath)
		if err != nil {
			return nil, err
		}
		var content []byte
		content, err = io.ReadAll(file)
		rawContent = string(content)

		// if os.File is been read by io.ReadAll, then proto.NewParser can't read it, so use
		// os.Open twice
		file, err = os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		parser = proto.NewParser(file)
	} else if conf.Task == TASK_CONTENT_PROTO2THRIFT {
		rd := strings.NewReader(conf.RawContent)

		rawContent = conf.RawContent
		parser = proto.NewParser(rd)
	}

	definition, err := parser.Parse()
	if err != nil {
		return
	}

	res = &ThriftGenerator{
		conf:       conf,
		def:        definition,
		file:       file,
		rawContent: rawContent,
	}
	return
}

func (g *ThriftGenerator) Generate() (err error) {
	if err = g.parse(); err != nil {
		return
	}
	if err = g.sink(); err != nil {
		return
	}
	return
}

// generate thriftAST from proto ast
func (g *ThriftGenerator) parse() (err error) {
	g.thriftAST = &thrift.Thrift{
		Enums:    make(map[string]*thrift.Enum),
		Structs:  make(map[string]*thrift.Struct),
		Services: make(map[string]*thrift.Service),
	}
	proto.Walk(
		g.def,
		proto.WithService(g.handleService),
		proto.WithMessage(g.handleMessage),
		proto.WithEnum(g.handleEnum),
	)
	return
}

// write thrift code from thriftAST to output
func (g *ThriftGenerator) sink() (err error) {
	g.sinkEnum()
	g.sinkStruct()
	g.sinkService()

	if g.conf.OutputPath != "" {
		var file *os.File
		file, err = os.Create(g.conf.OutputPath)
		if err != nil {
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

func (g *ThriftGenerator) handleService(s *proto.Service) {
	methodMap := make(map[string]*thrift.Method)
	g.thriftAST.Services[s.Name] = &thrift.Service{
		Name:    s.Name,
		Methods: methodMap,
	}
	for _, ele := range s.Elements {
		field := ele.(*proto.RPC)
		name := field.Name
		args := []*thrift.Field{
			// since protobuf rpc method request argument dont have name, we use a default name 'req'
			{
				ID:   1,
				Name: "req",
				Type: &thrift.Type{
					Name: field.RequestType,
				},
			},
		}
		methodMap[name] = &thrift.Method{
			Name:      name,
			Arguments: args,
			ReturnType: &thrift.Type{
				Name: field.ReturnsType,
			},
		}
	}
}

func (g *ThriftGenerator) handleEnum(s *proto.Enum) {
	valueMap := make(map[string]*thrift.EnumValue)
	g.thriftAST.Enums[s.Name] = &thrift.Enum{
		Name:   s.Name,
		Values: valueMap,
	}

	for _, ele := range s.Elements {
		field := ele.(*proto.EnumField)
		name := field.Name
		valueMap[name] = &thrift.EnumValue{
			Name:  name,
			Value: field.Integer,
		}
	}
}

func (g *ThriftGenerator) handleMessage(m *proto.Message) {
	fields := []*thrift.Field{}
	g.thriftAST.Structs[m.Name] = &thrift.Struct{
		Name:   m.Name,
		Fields: fields,
	}

	for _, ele := range m.Elements {
		var field *thrift.Field

		// handle fields except for map
		mes, ok := ele.(*proto.NormalField)
		if ok {
			field = &thrift.Field{
				ID:   mes.Sequence,
				Name: mes.Name,
			}

			if mes.Repeated {
				t, err := g.typeConverter(mes.Type)
				if err != nil {
					logger.Error(err)
					continue
				}
				field.Type = &thrift.Type{
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
				field = &thrift.Field{
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

				field.Type = &thrift.Type{
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

func (g *ThriftGenerator) typeConverter(t string) (res *thrift.Type, err error) {
	res, err = g.basicTypeConverter(t)
	if err != nil {
		// if t is not a basic type, then we should convert its case, same as name
		res = &thrift.Type{
			Name: utils.CaseConvert(g.conf.NameCase, t),
		}
		return res, nil
	}
	return
}

func (g *ThriftGenerator) basicTypeConverter(t string) (res *thrift.Type, err error) {
	switch t {
	case "string":
		res = &thrift.Type{
			Name: "string",
		}
	case "int64":
		res = &thrift.Type{
			Name: "i64",
		}
	case "int32":
		res = &thrift.Type{
			Name: "i32",
		}
	case "float", "double":
		res = &thrift.Type{
			Name: "double",
		}
	case "bool":
		res = &thrift.Type{
			Name: "bool",
		}
	case "bytes":
		res = &thrift.Type{
			Name: "binary",
		}
	default:
		err = fmt.Errorf("Invalid basic type %s", t)
	}
	return
}

func (g *ThriftGenerator) sinkService() {
	for _, s := range g.thriftAST.Services {
		name := utils.CaseConvert(g.conf.NameCase, s.Name)
		g.thriftContent.WriteString(fmt.Sprintf("service %s {\n", name))
		for _, m := range s.Methods {
			name := utils.CaseConvert(g.conf.NameCase, m.Name)
			g.writeIndent()
			g.thriftContent.WriteString(
				fmt.Sprintf(
					"%s %s (%d: %s %s);\n",
					m.ReturnType.String(),
					name,
					m.Arguments[0].ID,
					m.Arguments[0].Type.String(),
					utils.CaseConvert(g.conf.NameCase, m.Arguments[0].Name),
				),
			)
		}
		g.thriftContent.WriteString("}\n")
	}
}

func (g *ThriftGenerator) sinkEnum() {
	for _, enum := range g.thriftAST.Enums {
		name := utils.CaseConvert(g.conf.NameCase, enum.Name)
		g.thriftContent.WriteString(fmt.Sprintf("enum %s {\n", name))
		// since for-range map is random-ordered, we need to sort first, then write
		valueSlice := []*thrift.EnumValue{}
		for _, value := range enum.Values {
			valueSlice = append(valueSlice, value)
		}
		sort.Slice(valueSlice, func(i, j int) bool {
			return valueSlice[i].Value < valueSlice[j].Value
		})

		for _, field := range valueSlice {
			fieldName := utils.CaseConvert(g.conf.FieldCase, field.Name)
			g.writeIndent()
			g.thriftContent.WriteString(fmt.Sprintf("%s = %d\n", fieldName, field.Value))
		}
		g.thriftContent.WriteString("}\n")
	}
}

func (g *ThriftGenerator) sinkStruct() {
	for _, sct := range g.thriftAST.Structs {
		name := utils.CaseConvert(g.conf.NameCase, sct.Name)
		g.thriftContent.WriteString(fmt.Sprintf("struct %s {\n", name))

		for _, field := range sct.Fields {
			typeName := field.Type.String()
			fieldName := utils.CaseConvert(g.conf.FieldCase, field.Name)
			g.writeIndent()
			g.thriftContent.WriteString(fmt.Sprintf("%d: %s %s;\n", field.ID, typeName, fieldName))
		}

		g.thriftContent.WriteString("}\n")
	}
}

func (g *ThriftGenerator) writeIndent() {
	if g.conf.UseSpaceIndent {
		spaceCount, _ := strconv.Atoi(g.conf.IndentSpace)
		for i := 0; i < spaceCount; i++ {
			g.thriftContent.WriteString(" ")
		}
	} else {
		g.thriftContent.WriteString("	")
	}
}

package utils

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
)

func CaseConvert(strCase string, str string) (res string) {
	var name string
	var strItems []string
	// if str is prefixed with package name, e.g admin.EnumName, we should ignore the package name
	if strings.Contains(str, ".") {
		strItems = strings.Split(str, ".")
		name, strItems = strItems[len(strItems)-1], strItems[:len(strItems)-1]
	} else {
		name = str
	}
	switch strCase {
	case "camelCase":
		res = strcase.ToLowerCamel(name)
	case "snakeCase":
		res = strcase.ToSnake(name)
	case "kababCase":
		res = strcase.ToKebab(name)
	case "pascalCase":
		res = strcase.ToCamel(name)
	case "screamingSnakeCase":
		res = strcase.ToScreamingSnake(name)
	}

	if strings.Contains(str, ".") {
		res = fmt.Sprintf("%s.%s", strings.Join(strItems, "."), res)
	}
	return
}

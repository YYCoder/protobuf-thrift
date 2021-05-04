package utils

import "github.com/iancoleman/strcase"

func CaseConvert(strCase string, str string) (res string) {
	switch strCase {
	case "camelCase":
		res = strcase.ToLowerCamel(str)
	case "snakeCase":
		res = strcase.ToSnake(str)
	case "kababCase":
		res = strcase.ToKebab(str)
	case "pascalCase":
		res = strcase.ToCamel(str)
	case "screamingSnakeCase":
		res = strcase.ToScreamingSnake(str)
	}
	return
}

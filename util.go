package goas

import (
	"bufio"
	"fmt"
	"go/ast"
	"log"
	"os"
	"strings"
)

func isInStringList(list []string, s string) bool {
	for i, _ := range list {
		if list[i] == s {
			return true
		}
	}
	return false
}

// Find 'package main' and 'func main()'
func isMainFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var isMainPackage, hasMainFunc bool

	bs := bufio.NewScanner(f)
	for bs.Scan() {
		l := bs.Text()
		if !isMainPackage && strings.HasPrefix(l, "package main") {
			isMainPackage = true
		}
		if !hasMainFunc && strings.HasPrefix(l, "func main()") {
			hasMainFunc = true
		}
		if isMainPackage && hasMainFunc {
			break
		}
	}
	if bs.Err() != nil {
		log.Fatal(bs.Err())
	}

	return isMainPackage && hasMainFunc
}

var basicTypes = map[string]bool{
	"bool":       true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"float32":    true,
	"float64":    true,
	"string":     true,
	"complex64":  true,
	"complex128": true,
	"byte":       true,
	"rune":       true,
	"uintptr":    true,
	"error":      true,
	// "Time":      true,
	// "file":      true,
	// "undefined": true,
}

func isBasicType(typeName string) bool {
	_, ok := basicTypes[typeName]
	return ok || strings.Contains(typeName, "map") || strings.Contains(typeName, "interface")
}

var basicTypesOASTypes = map[string]string{
	"bool":    "boolean",
	"uint":    "integer",
	"uint8":   "integer",
	"uint16":  "integer",
	"uint32":  "integer",
	"uint64":  "integer",
	"int":     "integer",
	"int8":    "integer",
	"int16":   "integer",
	"int32":   "integer",
	"int64":   "integer",
	"float32": "number",
	"float64": "number",
	"string":  "string",
	// "file":    "formData",
}

func isBasicTypeOASType(typeName string) bool {
	_, ok := basicTypesOASTypes[typeName]
	return ok || strings.Contains(typeName, "map") || strings.Contains(typeName, "interface")
}

var basicTypesOASFormats = map[string]string{
	"bool":    "boolean",
	"uint":    "integer",
	"uint8":   "int64",
	"uint16":  "int64",
	"uint32":  "int64",
	"uint64":  "int64",
	"int":     "int64",
	"int8":    "int64",
	"int16":   "int64",
	"int32":   "int64",
	"int64":   "int64",
	"float32": "float",
	"float64": "double",
	"string":  "string",
}

var typeDefTranslations = map[string]string{}

var modelNamesPackageNames = map[string]string{}

func referenceLink(name string) string {
	if strings.HasPrefix(name, "#/schemas/") {
		return name
	}

	return "#/components/schemas/" + name
}

func getTypeAsString(fieldType interface{}) string {
	astArrayType, ok := fieldType.(*ast.ArrayType)
	if ok {
		// log.Printf("arrayType: %#v\n", astArrayType)
		return fmt.Sprintf("[]%v", getTypeAsString(astArrayType.Elt))
	}

	astMapType, ok := fieldType.(*ast.MapType)
	if ok {
		// log.Printf("astMapType: %#v\n", astMapType)
		return fmt.Sprintf("map[]%v", getTypeAsString(astMapType.Value))
	}

	_, ok = fieldType.(*ast.InterfaceType)
	if ok {
		return "interface"
	}

	astStarExpr, ok := fieldType.(*ast.StarExpr)
	if ok {
		// log.Printf("Get type as string (star expression)! %#v, type: %s\n", astStarExpr.X, fmt.Sprint(astStarExpr.X))
		return fmt.Sprint(astStarExpr.X)
	}

	astSelectorExpr, ok := fieldType.(*ast.SelectorExpr)
	if ok {
		// log.Printf("Get type as string(selector expression)! X: %#v , Sel: %#v, type %s\n", astSelectorExpr.X, astSelectorExpr.Sel, realType)
		packageNameIdent, _ := astSelectorExpr.X.(*ast.Ident)
		return packageNameIdent.Name + "." + astSelectorExpr.Sel.Name
	}

	// log.Printf("Get type as string(no star expression)! %#v , type: %s\n", fieldType, fmt.Sprint(fieldType))
	return fmt.Sprint(fieldType)
}

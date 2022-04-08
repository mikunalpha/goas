package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

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

func getModuleNameFromGoMod(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	moduleName := ""

	bs := bufio.NewScanner(f)
	for bs.Scan() {
		l := strings.TrimSpace(bs.Text())
		if strings.HasPrefix(l, "module") {
			moduleName = strings.TrimSpace(strings.TrimPrefix(l, "module"))
			break
		}
	}
	// if bs.Err() != nil {
	// 	return ""
	// }

	return moduleName
}

func isInStringList(list []string, s string) bool {
	for i, _ := range list {
		if list[i] == s {
			return true
		}
	}
	return false
}

func cleanBrackets(s string) string {
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return s[1 : len(s)-1]
	}
	return s
}

func isTerminal(s string) bool {
	cleanString := cleanBrackets(s)
	_, ok := TerminalTypes[cleanString]
	return ok
}

func isComplex(s string) bool {
	cleanString := cleanBrackets(s)
	_, ok := ComplexTypes[cleanString]
	return ok
}

var TerminalTypes = map[string]bool{
	"string":  true,
	"number":  true,
	"integer": true,
	"boolean": true,
}

var ComplexTypes = map[string]bool{
	"array":  true,
	"object": true,
}

var basicGoTypes = map[string]bool{
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
}

func isBasicGoType(typeName string) bool {
	_, ok := basicGoTypes[typeName]
	return ok
}

func isComplexGoType(typeName string) bool {
	return strings.HasPrefix(typeName, "[]") || strings.HasPrefix(typeName, "map[]")
}

var goTypesOASTypes = map[string]string{
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
}

func isGoTypeOASType(typeName string) bool {
	_, ok := goTypesOASTypes[typeName]
	return ok
}

var goTypesOASFormats = map[string]string{
	"bool":    "boolean",
	"uint":    "int64",
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

// var typeDefTranslations = map[string]string{}

// var modelNamesPackageNames = map[string]string{}

func addSchemaRefLinkPrefix(name string) string {
	if strings.HasPrefix(name, "#/components/schemas/") {
		return replaceBackslash(name)
	}
	return replaceBackslash("#/components/schemas/" + name)
}

func trimeSchemaRefLinkPrefix(ref string) string {
	return strings.TrimPrefix(ref, "#/components/schemas/")
}

func genSchemeaObjectID(pkgName, typeName string) string {
	typeNameParts := strings.Split(typeName, ".")
	pkgName = replaceBackslash(pkgName)
	return strings.Join(append(strings.Split(pkgName, "/"), typeNameParts[len(typeNameParts)-1]), ".")
}

func replaceBackslash(origin string) string {
	return strings.ReplaceAll(origin, "\\", "/")
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/mikunalpha/goas/go-module"
)

type parser struct {
	ModulePath string
	ModuleName string

	MainFilePath string

	GoModFilePath string

	RequirePkgs []pkg

	GoModCachePath string

	OASSpec *OASSpecObject

	TypeDefinitions map[string]map[string]*ast.TypeSpec
	PkgCache        map[string]map[string]*ast.Package
	PkgPathCache    map[string]string
	PkgImports      map[string]map[string][]string

	Debug bool
}

type pkg struct {
	Name string
	Path string
}

func newParser(modulePath, mainFilePath string, debug bool) (*parser, error) {
	p := &parser{
		TypeDefinitions: map[string]map[string]*ast.TypeSpec{},
		PkgCache:        map[string]map[string]*ast.Package{},
		PkgPathCache:    map[string]string{},
		PkgImports:      map[string]map[string][]string{},
		Debug:           debug,
	}

	// check modulePath is exist
	modulePath, _ = filepath.Abs(modulePath)
	moduleInfo, err := os.Stat(modulePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", modulePath, err)
	}
	if !moduleInfo.IsDir() {
		return nil, fmt.Errorf("modulePath should be a directory")
	}
	p.ModulePath = modulePath
	p.debugf("module path: %s", p.ModulePath)

	// check go.mod file is exist
	goModFilePath := filepath.Join(modulePath, "go.mod")
	goModFileInfo, err := os.Stat(goModFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", goModFilePath, err)
	}
	if goModFileInfo.IsDir() {
		return nil, fmt.Errorf("%s should be a file", goModFilePath)
	}
	p.GoModFilePath = goModFilePath
	p.debugf("go.mod file path: %s", p.GoModFilePath)

	// check mainFilePath is exist
	if mainFilePath == "" {
		fns, err := filepath.Glob(filepath.Join(modulePath, "*.go"))
		if err != nil {
			return nil, err
		}
		for _, fn := range fns {
			if isMainFile(fn) {
				mainFilePath = fn
				break
			}
		}
	} else {
		mainFileInfo, err := os.Stat(mainFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, err
			}
			return nil, fmt.Errorf("cannot get information of %s: %s", mainFilePath, err)
		}
		if mainFileInfo.IsDir() {
			return nil, fmt.Errorf("mainFilePath should not be a directory")
		}
	}
	p.MainFilePath = mainFilePath
	p.debugf("main file path: %s", p.MainFilePath)

	// get module name from go.mod file
	moduleName := getModuleNameFromGoMod(goModFilePath)
	if moduleName == "" {
		return nil, fmt.Errorf("cannot get module name from %s", goModFileInfo)
	}
	p.ModuleName = moduleName
	p.debugf("module name: %s", p.ModuleName)

	// check go module cache path is exist ($GOPATH/pkg/mod)
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		user, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("cannot get current user: %s", err)
		}
		goPath = filepath.Join(user.HomeDir, "go")
	}
	goModCachePath := filepath.Join(goPath, "pkg", "mod")
	goModCacheInfo, err := os.Stat(goModCachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", goModCachePath, err)
	}
	if !goModCacheInfo.IsDir() {
		return nil, fmt.Errorf("%s should be a directory", goModCachePath)
	}
	p.GoModCachePath = goModCachePath
	p.debugf("go module cache path: %s", p.GoModCachePath)

	return p, nil
}

func (p *parser) CreateOASFile(path string) error {
	// parse basic info
	err := p.parseInfo()
	if err != nil {
		return err
	}

	// parse sub-package
	err = p.parseModule()
	if err != nil {
		return err
	}

	// parse go.mod info
	err = p.parseGoMod()
	if err != nil {
		return err
	}

	// parse APIs info
	err = p.parseAPIs()
	if err != nil {
		return err
	}

	fd, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Can not create the file %s: %v", path, err)
	}
	defer fd.Close()

	output, err := json.MarshalIndent(p.OASSpec, "", "  ")
	if err != nil {
		return err
	}
	_, err = fd.WriteString(string(output))

	return err
}

func (p *parser) parseInfo() error {
	p.OASSpec = &OASSpecObject{
		OpenAPI: OpenAPIVersion,
		Servers: []*ServerObject{{URL: "/"}},
		Info:    &InfoObject{},
		Paths:   map[string]*PathItemObject{},
	}

	fileTree, err := goparser.ParseFile(token.NewFileSet(), p.MainFilePath, nil, goparser.ParseComments)
	if err != nil {
		return fmt.Errorf("Can not parse general API information: %v\n", err)
	}
	if fileTree.Comments != nil {
		for _, comment := range fileTree.Comments {
			for _, commentLine := range strings.Split(comment.Text(), "\n") {
				attribute := strings.ToLower(strings.Split(commentLine, " ")[0])
				value := strings.TrimSpace(commentLine[len(attribute):])
				if len(value) == 0 {
					continue
				}
				p.debug(commentLine)
				switch attribute {
				case "@version":
					p.OASSpec.Info.Version = value
				case "@title":
					p.OASSpec.Info.Title = value
				case "@description":
					p.OASSpec.Info.Description = value
				case "@termsofserviceurl":
					p.OASSpec.Info.TermsOfService = value
				case "@contactname":
					if p.OASSpec.Info.Contact == nil {
						p.OASSpec.Info.Contact = &ContactObject{}
					}
					p.OASSpec.Info.Contact.Name = value
				case "@contactemail":
					if p.OASSpec.Info.Contact == nil {
						p.OASSpec.Info.Contact = &ContactObject{}
					}
					p.OASSpec.Info.Contact.Email = value
				case "@contacturl":
					if p.OASSpec.Info.Contact == nil {
						p.OASSpec.Info.Contact = &ContactObject{}
					}
					p.OASSpec.Info.Contact.URL = value
				case "@licensename":
					if p.OASSpec.Info.License == nil {
						p.OASSpec.Info.License = &LicenseObject{}
					}
					p.OASSpec.Info.License.Name = value
				case "@licenseurl":
					if p.OASSpec.Info.License == nil {
						p.OASSpec.Info.License = &LicenseObject{}
					}
					p.OASSpec.Info.License.URL = value
				case "@server":
					fields := strings.Split(value, " ")
					s := &ServerObject{URL: fields[0], Description: value[len(fields[0]):]}
					p.OASSpec.Servers = append(p.OASSpec.Servers, s)
				}
			}
		}
	}

	if len(p.OASSpec.Servers) < 1 {
		p.OASSpec.Servers = append(p.OASSpec.Servers, &ServerObject{URL: "/", Description: "Default Server URL"})
	}

	if p.OASSpec.Info.Title == "" {
		return fmt.Errorf("info.title cannot not be empty")
	}
	if p.OASSpec.Info.Version == "" {
		return fmt.Errorf("info.version cannot not be empty")
	}
	for i := range p.OASSpec.Servers {
		if p.OASSpec.Servers[i].URL == "" {
			return fmt.Errorf("servers[%d].url cannot not be empty", i)
		}
	}

	return nil
}

func (p *parser) parseModule() error {
	walker := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			name := filepath.Join(p.ModuleName, strings.TrimPrefix(path, p.ModulePath))
			p.RequirePkgs = append(p.RequirePkgs, pkg{
				Name: name,
				Path: path,
			})
			p.PkgPathCache[name] = path
		}
		return nil
	}
	filepath.Walk(p.ModulePath, walker)
	return nil
}

func (p *parser) parseGoMod() error {
	b, err := ioutil.ReadFile(p.GoModFilePath)
	if err != nil {
		return err
	}
	goMod, err := module.Parse(b)
	if err != nil {
		return err
	}
	for i := range goMod.Requires {
		pathRunes := []rune{}
		for _, v := range goMod.Requires[i].Path {
			if !unicode.IsUpper(v) {
				pathRunes = append(pathRunes, v)
				continue
			}
			pathRunes = append(pathRunes, '!')
			pathRunes = append(pathRunes, unicode.ToLower(v))
		}
		path := filepath.Join(p.GoModCachePath, string(pathRunes)+"@"+goMod.Requires[i].Version)
		p.RequirePkgs = append(p.RequirePkgs, pkg{
			Name: goMod.Requires[i].Path,
			Path: path,
		})
		p.PkgPathCache[goMod.Requires[i].Path] = path
	}
	if p.Debug {
		for i := range p.RequirePkgs {
			p.debug(p.RequirePkgs[i].Name)
			p.debug(p.RequirePkgs[i].Path)
		}
	}

	return nil
}

func (p *parser) parseAPIs() error {
	err := p.parseTypeDefinitions()
	if err != nil {
		return err
	}

	err = p.parsePaths()
	if err != nil {
		return err
	}

	return nil
}

func (p *parser) parseTypeDefinitions() error {
	for i := range p.RequirePkgs {
		pkgPath := p.RequirePkgs[i].Path

		_, ok := p.TypeDefinitions[pkgPath]
		if !ok {
			p.TypeDefinitions[pkgPath] = map[string]*ast.TypeSpec{}
		}
		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			return fmt.Errorf("Parse of %s package cause error: %s", pkgPath, err)
		}
		for _, astPackage := range astPkgs {
			for _, astFile := range astPackage.Files {
				for _, astDeclaration := range astFile.Decls {
					if generalDeclaration, ok := astDeclaration.(*ast.GenDecl); ok && generalDeclaration.Tok == token.TYPE {
						for _, astSpec := range generalDeclaration.Specs {
							if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
								p.TypeDefinitions[pkgPath][typeSpec.Name.String()] = typeSpec
							}
						}
					}
				}
			}
		}

		p.parseImportStatements(pkgPath)
	}

	return nil
}

func (p *parser) parseImportStatements(pkgPath string) map[string]bool {
	imports := map[string]bool{}
	astPackages, err := p.getPkgAst(pkgPath)
	if err != nil {
		return nil
	}

	p.PkgImports[pkgPath] = map[string][]string{}
	for _, astPackage := range astPackages {
		for _, astFile := range astPackage.Files {
			for _, astImport := range astFile.Imports {
				importedPackageName := strings.Trim(astImport.Path.Value, "\"")
				realPath := ""
				for i := range p.RequirePkgs {
					if p.RequirePkgs[i].Name == importedPackageName {
						realPath = p.RequirePkgs[i].Path
					}
				}
				if _, ok := p.TypeDefinitions[realPath]; !ok {
					imports[importedPackageName] = true
				}

				// Deal with alias of imported package
				var importedPackageAlias string
				if astImport.Name != nil && astImport.Name.Name != "." && astImport.Name.Name != "_" {
					importedPackageAlias = astImport.Name.Name
				} else {
					importPath := strings.Split(importedPackageName, "/")
					importedPackageAlias = importPath[len(importPath)-1]
				}

				isExists := false
				for _, v := range p.PkgImports[pkgPath][importedPackageAlias] {
					if v == importedPackageName {
						isExists = true
					}
				}
				if !isExists {
					p.PkgImports[pkgPath][importedPackageAlias] = append(p.PkgImports[pkgPath][importedPackageAlias], importedPackageName)
				}
			}
		}
	}
	return imports
}

func (p *parser) getPkgAst(pkgPath string) (map[string]*ast.Package, error) {
	if cache, ok := p.PkgCache[pkgPath]; ok {
		return cache, nil
	}
	astPackages, err := goparser.ParseDir(token.NewFileSet(), pkgPath, ignoreFileFilter, goparser.ParseComments)
	if err != nil {
		return nil, err
	}
	p.PkgCache[pkgPath] = astPackages

	return astPackages, nil
}

func ignoreFileFilter(info os.FileInfo) bool {
	name := info.Name()
	return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func (p *parser) parsePaths() error {
	for i := range p.RequirePkgs {
		pkgPath := p.RequirePkgs[i].Path
		pkgName := p.RequirePkgs[i].Name
		p.debugf("parsePaths with %s", pkgPath)

		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			return fmt.Errorf("Parse of %s package cause error: %s", pkgPath, err)
		}
		for _, astPackage := range astPkgs {
			for _, astFile := range astPackage.Files {
				for _, astDescription := range astFile.Decls {
					switch astDeclaration := astDescription.(type) {
					case *ast.FuncDecl:
						operation := &OperationObject{
							Responses: map[string]*ResponseObject{},
						}
						if astDeclaration.Doc != nil && astDeclaration.Doc.List != nil {
							for _, comment := range astDeclaration.Doc.List {
								err := p.parseOperation(operation, pkgPath, pkgName, comment.Text)
								if err != nil {
									return fmt.Errorf("can not parse comment for function: %v, package: %v, got error: %v", astDeclaration.Name.String(), pkgPath, err)
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (p *parser) parseOperation(operation *OperationObject, pkgPath, pkgName, comment string) error {
	if !strings.HasPrefix(pkgPath, p.ModulePath) {
		// ignore this pkgPath
		// p.debugf("parseOperation ignores %s", pkgPath)
		return nil
	}
	commentLine := strings.TrimSpace(strings.TrimLeft(comment, "//"))
	if len(commentLine) == 0 {
		return nil
	}

	attribute := strings.Fields(commentLine)[0]
	switch strings.ToLower(attribute) {
	case "@title":
		operation.Summary = strings.TrimSpace(commentLine[len(attribute):])
	case "@description":
		operation.Description = strings.TrimSpace(commentLine[len(attribute):])
	case "@param":
		p.debugf("parseOperation parse comment %s", comment)
		err := p.parseParamComment(pkgName, operation, strings.TrimSpace(commentLine[len(attribute):]))
		if err != nil {
			return err
		}
	case "@success", "@failure":
		err := p.parseResponseComment(pkgName, operation, strings.TrimSpace(commentLine[len(attribute):]))
		if err != nil {
			return err
		}
	case "@resource":
		resource := strings.TrimSpace(commentLine[len(attribute):])
		if resource == "" {
			resource = "others"
		}
		if !isInStringList(operation.Tags, resource) {
			operation.Tags = append(operation.Tags, resource)
		}
	case "@router":
		err := p.parseRouteComment(operation, commentLine)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *parser) parseParamComment(pkgName string, operation *OperationObject, paramString string) error {
	re := regexp.MustCompile(`([-\w]+)[\s]+([\w]+)[\s]+([\w./]+)[\s]+([\w]+)[\s]+"([^"]+)"`)
	matches := re.FindStringSubmatch(paramString)
	if len(matches) != 6 {
		return fmt.Errorf("Can not parse param comment \"%s\", skipped", paramString)
	}
	parameter := &ParameterObject{}
	parameter.Name = matches[1]
	parameter.In = matches[2]

	if parameter.In == "file" || parameter.In == "form" {
		ContentTypeForm := "multipart/form-data"

		requiredText := strings.ToLower(matches[4])

		if operation.RequestBody == nil {
			operation.RequestBody = &RequestBodyObject{
				Content: map[string]*MediaTypeObject{
					ContentTypeForm: &MediaTypeObject{
						Schema: &SchemaObject{
							Type:       "object",
							Properties: map[string]interface{}{},
						},
					},
				},
				Required: (requiredText == "true" || requiredText == "required"),
			}
		}

		if parameter.In == "file" {
			operation.RequestBody.Content[ContentTypeForm].Schema.Properties[parameter.Name] = &SchemaObject{
				Type:   "string",
				Format: "binary",
			}
		} else {
			typeName := matches[3]
			if isBasicTypeOASType(typeName) {
				operation.RequestBody.Content[ContentTypeForm].Schema.Properties[parameter.Name] = &SchemaObject{
					Type:   basicTypesOASTypes[typeName],
					Format: basicTypesOASFormats[typeName],
				}
			}
		}

		return nil
	}

	typeName, err := p.registerType(pkgName, matches[3])
	if err != nil {
		return err
	}

	if parameter.In != "body" {
		if isBasicTypeOASType(typeName) {
			parameter.Schema = &SchemaObject{
				Type:   basicTypesOASTypes[typeName],
				Format: basicTypesOASFormats[typeName],
			}
		} else {
			_, ok := p.OASSpec.Components.Schemas[typeName]
			if ok {
				parameter.Schema = &SchemaObject{
					Ref: referenceLink(typeName),
				}
			} else {
				parameter.Schema = &SchemaObject{
					Type: typeName,
				}
			}
		}
		requiredText := strings.ToLower(matches[4])
		parameter.Required = (requiredText == "true" || requiredText == "required")
		if parameter.In == "path" {
			parameter.Required = true
		}
		parameter.Description = matches[5]
		operation.Parameters = append(operation.Parameters, parameter)

		return nil
	}

	if operation.RequestBody == nil {
		operation.RequestBody = &RequestBodyObject{
			Content:  map[string]*MediaTypeObject{},
			Required: true,
		}
	}
	operation.RequestBody.Content[ContentTypeJson] = &MediaTypeObject{}
	_, ok := p.OASSpec.Components.Schemas[typeName]
	if ok {
		operation.RequestBody.Content[ContentTypeJson].Schema = &SchemaObject{
			Ref: referenceLink(typeName),
		}
	} else {
		operation.RequestBody.Content[ContentTypeJson].Schema = &SchemaObject{
			Type: strings.Trim(matches[2], "{}"),
		}
	}
	if matches[2] == "{array}" {
		operation.RequestBody.Content[ContentTypeJson].Schema = &SchemaObject{
			Type: "array",
			Items: &ReferenceObject{
				Ref: referenceLink(typeName),
				// Ref: typeName,
			},
		}
	} else if operation.RequestBody.Content[ContentTypeJson].Schema.Ref == "" {
		operation.RequestBody.Content[ContentTypeJson].Schema.Type = typeName
	}

	return nil
}

func (p *parser) parseRouteComment(operation *OperationObject, commentLine string) error {
	sourceString := strings.TrimSpace(commentLine[len("@Router"):])

	re := regexp.MustCompile(`([\w\.\/\-{}]+)[^\[]+\[([^\]]+)`)
	var matches []string

	matches = re.FindStringSubmatch(sourceString)
	if len(matches) != 3 {
		return fmt.Errorf("Can not parse router comment \"%s\", skipped", commentLine)
	}

	_, ok := p.OASSpec.Paths[matches[1]]
	if !ok {
		p.OASSpec.Paths[matches[1]] = &PathItemObject{}
	}

	switch strings.ToUpper(matches[2]) {
	case "GET":
		if p.OASSpec.Paths[matches[1]].Get == nil {
			p.OASSpec.Paths[matches[1]].Get = operation
		}
	case "POST":
		if p.OASSpec.Paths[matches[1]].Post == nil {
			p.OASSpec.Paths[matches[1]].Post = operation
		}
	case "PATCH":
		if p.OASSpec.Paths[matches[1]].Patch == nil {
			p.OASSpec.Paths[matches[1]].Patch = operation
		}
	case "PUT":
		if p.OASSpec.Paths[matches[1]].Put == nil {
			p.OASSpec.Paths[matches[1]].Put = operation
		}
	case "DELETE":
		if p.OASSpec.Paths[matches[1]].Delete == nil {
			p.OASSpec.Paths[matches[1]].Delete = operation
		}
	case "OPTIONS":
		if p.OASSpec.Paths[matches[1]].Options == nil {
			p.OASSpec.Paths[matches[1]].Options = operation
		}
	case "HEAD":
		if p.OASSpec.Paths[matches[1]].Head == nil {
			p.OASSpec.Paths[matches[1]].Head = operation
		}
	case "TRACE":
		if p.OASSpec.Paths[matches[1]].Trace == nil {
			p.OASSpec.Paths[matches[1]].Trace = operation
		}
	}

	return nil
}

func (p *parser) parseResponseComment(pkgName string, operation *OperationObject, commentLine string) error {
	re := regexp.MustCompile(`([\d]+)[\s]+([\w\{\}]+)[\s]+([\w\-\.\/]+)[^"]*(.*)?`)
	var matches []string

	matches = re.FindStringSubmatch(commentLine)
	if len(matches) != 5 {
		return fmt.Errorf("Can not parse response comment \"%s\", skipped", commentLine)
	}

	var response *ResponseObject
	var code int
	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return errors.New("Success http code must be int")
	} else {
		operation.Responses[fmt.Sprint(code)] = &ResponseObject{
			Content: map[string]*MediaTypeObject{},
		}
		response = operation.Responses[fmt.Sprint(code)]
		response.Content[ContentTypeJson] = &MediaTypeObject{}
	}
	response.Description = strings.Trim(matches[4], "\"")

	typeName, err := p.registerType(pkgName, matches[3])
	if err != nil {
		return err
	}

	_, ok := p.OASSpec.Components.Schemas[typeName]
	if ok {
		response.Content[ContentTypeJson].Schema = &SchemaObject{
			Ref: referenceLink(typeName),
			// Ref: typeName,
		}
	} else {
		response.Content[ContentTypeJson].Schema = &SchemaObject{
			Type: strings.Trim(matches[2], "{}"),
		}
	}

	if matches[2] == "{array}" {
		response.Content[ContentTypeJson].Schema = &SchemaObject{
			Type: "array",
			Items: &ReferenceObject{
				Ref: referenceLink(typeName),
				// Ref: typeName,
			},
		}
	} else if response.Content[ContentTypeJson].Schema.Ref == "" {
		response.Content[ContentTypeJson].Schema.Type = typeName
	}

	// output, err := json.MarshalIndent(response, "", "  ")
	// fmt.Println(string(output))

	return nil
}

func (p *parser) registerType(pkgName, typeName string) (string, error) {
	registerType := ""
	translation, ok := typeDefTranslations[typeName]
	if ok {
		registerType = translation
	} else if isBasicType(typeName) {
		registerType = typeName
	} else {
		model := &Model{}
		knownModelNames := map[string]bool{}
		innerModels, err := p.parseModel(model, typeName, pkgName, knownModelNames)
		if err != nil {
			return registerType, err
		}
		if translation, ok := typeDefTranslations[typeName]; ok {
			registerType = translation
		} else {
			registerType = model.Id
			if p.OASSpec.Components == nil {
				p.OASSpec.Components = &ComponentsOjbect{
					Schemas:    map[string]*SchemaObject{},
					Responses:  map[string]*ResponseObject{},
					Parameters: map[string]*ParameterObject{},
				}
			}
			_, ok := p.OASSpec.Components.Schemas[registerType]
			if !ok {
				p.OASSpec.Components.Schemas[registerType] = &SchemaObject{
					Type:       "object",
					Required:   model.Required,
					Properties: map[string]interface{}{},
				}
			}
			for k, v := range model.Properties {
				if v.Ref != "" {
					v.Type = ""
					v.Items = nil
					v.Format = ""
				}
				p.OASSpec.Components.Schemas[registerType].Properties[k] = v
			}
			for _, m := range innerModels {
				registerType := m.Id
				if m.Ref != "" {
					if _, ok := p.OASSpec.Components.Schemas[registerType]; !ok {
						p.OASSpec.Components.Schemas[registerType] = &SchemaObject{
							Ref: m.Ref,
						}
					}
					continue
				}
				if _, ok := p.OASSpec.Components.Schemas[registerType]; !ok {
					p.OASSpec.Components.Schemas[registerType] = &SchemaObject{
						Type:       "object",
						Required:   m.Required,
						Properties: map[string]interface{}{},
					}
				}
				for k, v := range m.Properties {
					if v.Ref != "" {
						v.Type = ""
						v.Items = nil
						v.Format = ""
					}
					if v.Example != nil {
						v.Ref = ""
						// fmt.Println(m.Id, k, "type", t)
						example, ok := v.Example.(string)
						if ok {
							if strings.HasPrefix(example, "{") {
								b, err := json.RawMessage(example).MarshalJSON()
								if err != nil {
									v.Example = "invalid example"
								} else {
									mapOfInterface := map[string]interface{}{}
									err := json.Unmarshal(b, &mapOfInterface)
									if err != nil {
										v.Example = "invalid example"
									} else {
										v.Example = mapOfInterface
									}
								}
							}
						}
					}
					p.OASSpec.Components.Schemas[registerType].Properties[k] = v
				}
			}
		}
	}

	return registerType, nil
}

type Model struct {
	Ref        string                    `json:"$ref,omitempty"`
	Id         string                    `json:"id,omitempty"`
	Required   []string                  `json:"required,omitempty"`
	Properties map[string]*ModelProperty `json:"properties,omitempty"`
}

type ModelProperty struct {
	Ref         string              `json:"$ref,omitempty"`
	Type        string              `json:"type,omitempty"`
	Description string              `json:"description,omitempty"`
	Format      string              `json:"format,omitempty"`
	Items       *ModelPropertyItems `json:"items,omitempty"`
	Example     interface{}         `json:"example,omitempty"`
}

type ModelPropertyItems struct {
	Ref  string `json:"$ref,omitempty"`
	Type string `json:"type,omitempty"`
}

func (p *parser) parseModel(m *Model, modelName string, pkgName string, knownModelNames map[string]bool) ([]*Model, error) {
	knownModelNames[modelName] = true
	astTypeSpec, modelPackage := p.findModelDefinition(modelName, pkgName)
	modelNameParts := strings.Split(modelName, ".")
	m.Id = strings.Join(append(strings.Split(modelPackage, "/"), modelNameParts[len(modelNameParts)-1]), ".")
	_, ok := modelNamesPackageNames[modelName]
	if !ok {
		modelNamesPackageNames[modelName] = m.Id
	}
	var innerModelList []*Model
	astTypeDef, ok := astTypeSpec.Type.(*ast.Ident)
	if ok {
		typeDefTranslations[m.Id] = astTypeDef.Name
		if modelRef, ok := modelNamesPackageNames[astTypeDef.Name]; ok {
			m.Ref = modelRef
		} else {
			var err error
			typeModel := &Model{}
			innerModelList, err = p.parseModel(typeModel, astTypeDef.Name, modelPackage, knownModelNames)
			if err != nil {
				p.debugf("parse model %s failed: %s\n", astTypeDef.Name, err)
			} else {
				innerModelList = append(innerModelList, typeModel)
			}
		}
		if modelRef, ok := modelNamesPackageNames[astTypeDef.Name]; ok {
			m.Ref = referenceLink(modelRef)
		}
	} else if astStructType, ok := astTypeSpec.Type.(*ast.StructType); ok {
		p.parseFieldList(m, astStructType.Fields.List, modelPackage)
		usedTypes := map[string]bool{}

		for _, property := range m.Properties {
			typeName := property.Type
			if isBasicType(property.Format) {
				typeName = property.Format
				// log.Printf("%s %+v", m.Id, *property)
			}
			if typeName == "array" {
				if property.Items.Type != "" {
					typeName = property.Items.Type
				} else {
					typeName = property.Items.Ref
				}
			}
			if translation, ok := typeDefTranslations[typeName]; ok {
				typeName = translation
			}
			if isBasicType(typeName) {
				if isBasicTypeOASType(typeName) {
					property.Format = basicTypesOASFormats[typeName]
					if property.Type != "array" {
						property.Type = basicTypesOASTypes[typeName]
					} else {
						if isBasicType(property.Items.Type) {
							if isBasicTypeOASType(property.Items.Type) {
								property.Items.Type = basicTypesOASTypes[property.Items.Type]
							}
						}
					}
				}
				continue
			}
			// if g.isImplementMarshalInterface(typeName) {
			// 	continue
			// }
			if _, exists := knownModelNames[typeName]; exists {
				// fmt.Println("@", typeName)
				if _, ok := modelNamesPackageNames[typeName]; ok {
					if translation, ok := typeDefTranslations[modelNamesPackageNames[typeName]]; ok {
						if isBasicType(translation) {
							if isBasicTypeOASType(translation) {
								// fmt.Println(modelNamesPackageNames[typeName], translation)
								property.Type = basicTypesOASTypes[translation]
							}
							continue
						}
					}
					if property.Type != "array" {
						property.Ref = referenceLink(modelNamesPackageNames[typeName])
						// property.Ref = modelNamesPackageNames[typeName]
					} else {
						property.Items.Ref = referenceLink(modelNamesPackageNames[typeName])
						// property.Items.Ref = modelNamesPackageNames[typeName]
					}
				}
				continue
			}

			usedTypes[typeName] = true
		}

		// log.Printf("Before parse inner model list: %#v\n (%s)", usedTypes, modelName)
		// innerModelList = []*Model{}

		for typeName := range usedTypes {
			typeModel := &Model{}
			if typeInnerModels, err := p.parseModel(typeModel, typeName, modelPackage, knownModelNames); err != nil {
				//log.Printf("Parse Inner Model error %#v \n", err)
				return nil, err
			} else {
				for _, property := range m.Properties {
					if property.Type == "array" {
						if property.Items.Ref == typeName {
							property.Items.Ref = referenceLink(typeModel.Id)
						}
					} else {
						if property.Type == typeName {
							if translation, ok := typeDefTranslations[modelNamesPackageNames[typeName]]; ok {
								if isBasicType(translation) {
									if isBasicTypeOASType(translation) {
										property.Type = basicTypesOASTypes[translation]
									}
									continue
								}
								// if indirectRef, ok := modelNamesPackageNames[translation]; ok {
								// 	property.Ref = referenceLink(indirectRef)
								// 	continue
								// }
							}
							property.Ref = referenceLink(typeModel.Id)
						} else {
							// fmt.Println(property.Type, "<>", typeName)
						}
					}
				}
				//log.Printf("Inner model %v parsed, parsing %s \n", typeName, modelName)
				if typeModel != nil {
					innerModelList = append(innerModelList, typeModel)
				}
				if typeInnerModels != nil && len(typeInnerModels) > 0 {
					innerModelList = append(innerModelList, typeInnerModels...)
				}
				//log.Printf("innerModelList: %#v\n, typeInnerModels: %#v, usedTypes: %#v \n", innerModelList, typeInnerModels, usedTypes)
			}
		}
		// log.Printf("After parse inner model list: %#v\n (%s)", usedTypes, modelName)
		// log.Fatalf("Inner model list: %#v\n", innerModelList)

	} else if astMapType, ok := astTypeSpec.Type.(*ast.MapType); ok {
		m.Properties = map[string]*ModelProperty{}
		mapProperty := &ModelProperty{}
		m.Properties["key"] = mapProperty

		typeName := fmt.Sprint(astMapType.Value)

		reInternalIndirect := regexp.MustCompile("&\\{(\\w*) <nil> (\\w*)\\}")
		typeName = string(reInternalIndirect.ReplaceAll([]byte(typeName), []byte("[]$2")))

		reInternalRepresentation := regexp.MustCompile("&\\{(\\w*) (\\w*)\\}")
		typeName = string(reInternalRepresentation.ReplaceAll([]byte(typeName), []byte("$1.$2")))

		if strings.HasPrefix(typeName, "[]") {
			mapProperty.Type = "array"
			p.setItemType(mapProperty, typeName[2:])
			// if is Unsupported item type of list, ignore this mapProperty
			if mapProperty.Items.Type == "undefined" {
				mapProperty = nil
			}
		} else if strings.HasPrefix(typeName, "map[]") {
			mapProperty.Type = "map"
		} else if typeName == "time.Time" {
			mapProperty.Type = "Time"
		} else {
			mapProperty.Type = typeName
		}
		typeName = mapProperty.Type

		// fmt.Println(mapProperty.Items.Type)

		if typeName == "array" {
			// mapProperty.Items = &ModelPropertyItems{}
			if mapProperty.Items.Type != "" {
				typeName = mapProperty.Items.Type
			} else {
				typeName = mapProperty.Items.Ref
			}
		}
		if translation, ok := typeDefTranslations[typeName]; ok {
			typeName = translation
		}

		if isBasicType(typeName) {
			if isBasicTypeOASType(typeName) {
				if mapProperty.Type != "array" {
					mapProperty.Format = basicTypesOASFormats[typeName]
					mapProperty.Type = basicTypesOASTypes[typeName]
				} else {
					if isBasicType(mapProperty.Items.Type) {
						if isBasicTypeOASType(mapProperty.Items.Type) {
							mapProperty.Items.Type = basicTypesOASTypes[mapProperty.Items.Type]
						}
					}
				}
			}
		} else {
			if _, exists := knownModelNames[typeName]; exists {
				// fmt.Println("@", typeName)
				if _, ok := modelNamesPackageNames[typeName]; ok {
					if translation, ok := typeDefTranslations[modelNamesPackageNames[typeName]]; ok {
						if isBasicType(translation) {
							if isBasicTypeOASType(translation) {
								// fmt.Println(modelNamesPackageNames[typeName], translation)
								mapProperty.Type = basicTypesOASTypes[translation]
							}
							// continue
						}
					}
					if mapProperty.Type != "array" {
						mapProperty.Ref = referenceLink(modelNamesPackageNames[typeName])
						// mapProperty.Ref = modelNamesPackageNames[typeName]
					} else {
						mapProperty.Items.Ref = referenceLink(modelNamesPackageNames[typeName])
						// mapProperty.Items.Ref = modelNamesPackageNames[typeName]
					}
				}
			}

			typeModel := &Model{}
			if typeInnerModels, err := p.parseModel(typeModel, typeName, modelPackage, knownModelNames); err != nil {
				//log.Printf("Parse Inner Model error %#v \n", err)
				return nil, err
			} else {
				for _, property := range m.Properties {
					if property.Type == "array" {
						if property.Items.Ref == typeName {
							property.Items.Ref = referenceLink(typeModel.Id)
						}
					} else {
						if property.Type == typeName {
							if translation, ok := typeDefTranslations[modelNamesPackageNames[typeName]]; ok {
								if isBasicType(translation) {
									if isBasicTypeOASType(translation) {
										property.Type = basicTypesOASTypes[translation]
									}
									continue
								}
								// if indirectRef, ok := modelNamesPackageNames[translation]; ok {
								// 	property.Ref = referenceLink(indirectRef)
								// 	continue
								// }
							}
							property.Ref = referenceLink(typeModel.Id)
						} else {
							// fmt.Println(property.Type, "<>", typeName)
						}
					}
				}
				//log.Printf("Inner model %v parsed, parsing %s \n", typeName, modelName)
				if typeModel != nil {
					innerModelList = append(innerModelList, typeModel)
				}
				if typeInnerModels != nil && len(typeInnerModels) > 0 {
					innerModelList = append(innerModelList, typeInnerModels...)
				}
				//log.Printf("innerModelList: %#v\n, typeInnerModels: %#v, usedTypes: %#v \n", innerModelList, typeInnerModels, usedTypes)
			}
		}

	}

	//log.Printf("ParseModel finished %s \n", modelName)
	return innerModelList, nil
}

func (p *parser) findModelDefinition(modelName string, pkgName string) (*ast.TypeSpec, string) {
	var model *ast.TypeSpec
	var modelPackage string

	modelNameParts := strings.Split(modelName, ".")

	// if no dot in name - it can be only model from current package
	if len(modelNameParts) == 1 {
		modelPackage = pkgName
		model = p.getModelDefinition(modelName, p.getPkgPathByPkgName(pkgName))
		if model == nil {
			fmt.Println(">>", pkgName, p.getPkgPathByPkgName(pkgName))
			fmt.Println(">>>", modelName)
			log.Fatalf("Can not find definition of %s model. Current package %s", modelName, pkgName)
		}
	} else {
		// // First try to assume what name is absolute
		absolutePackageName := strings.Join(modelNameParts[:len(modelNameParts)-1], "/")
		modelNameFromPath := modelNameParts[len(modelNameParts)-1]

		// for k := range p.TypeDefinitions {
		// 	p.debug("kkk ", k)
		// }

		modelPackage = absolutePackageName
		model = p.getModelDefinition(modelNameFromPath, absolutePackageName)
		if model == nil {
			// 	// Can not get model by absolute name.
			if len(modelNameParts) > 2 {
				log.Fatalf("Can not find definition of %s model. Name looks like absolute, but model not found in %s", modelNameFromPath, pkgName)
			}

			// Lets try to find it in imported packages
			imports, ok := p.PkgImports[p.getPkgPathByPkgName(pkgName)]
			if !ok {
				log.Fatalf("Can not find definition of %s model. Package %s dont import anything", modelNameFromPath, pkgName)
			}
			relativePackage, ok := imports[modelNameParts[0]]
			if !ok {
				log.Fatalf("Package %s is not imported to %s, Imported: %#v\n", modelNameParts[0], pkgName, imports)
			}

			var modelFound bool
			for _, packageName := range relativePackage {
				realPath := ""
				for i := range p.RequirePkgs {
					if p.RequirePkgs[i].Name == packageName {
						realPath = p.RequirePkgs[i].Path
					}
				}
				model = p.getModelDefinition(modelNameFromPath, realPath)
				if model != nil {
					modelPackage = packageName
					modelFound = true
					break
				}
			}
			if !modelFound {
				p.debug(imports[modelNameParts[0]])
				p.debug(p.TypeDefinitions)
				log.Fatalf("Can not find definition of %s model in package %s", modelNameFromPath, relativePackage)
			}
		}
	}

	return model, modelPackage
}

func (p *parser) getModelDefinition(model string, pkgName string) *ast.TypeSpec {
	packageModels, ok := p.TypeDefinitions[pkgName]
	if !ok {
		return nil
	}
	astTypeSpec, _ := packageModels[model]
	return astTypeSpec
}

func (p *parser) parseFieldList(m *Model, fieldList []*ast.Field, modelPackage string) {
	if fieldList == nil {
		return
	}
	m.Properties = map[string]*ModelProperty{}
	for _, field := range fieldList {
		p.parseModelProperty(m, field, modelPackage)
	}
}

func (p *parser) parseModelProperty(m *Model, field *ast.Field, modelPackage string) {
	var name string
	var innerModel *Model

	property := &ModelProperty{}

	typeAsString := getTypeAsString(field.Type)
	// log.Printf("Get type as string %s \n", typeAsString)

	reInternalIndirect := regexp.MustCompile("&\\{(\\w*) <nil> (\\w*)\\}")
	typeAsString = string(reInternalIndirect.ReplaceAll([]byte(typeAsString), []byte("[]$2")))

	// Sometimes reflection reports an object as "&{foo Bar}" rather than just "foo.Bar"
	// The next 2 lines of code normalize them to foo.Bar
	reInternalRepresentation := regexp.MustCompile("&\\{(\\w*) (\\w*)\\}")
	typeAsString = string(reInternalRepresentation.ReplaceAll([]byte(typeAsString), []byte("$1.$2")))

	// fmt.Println(m.Id, field.Names, typeAsString)

	if strings.HasPrefix(typeAsString, "[]") {
		property.Type = "array"
		p.setItemType(property, typeAsString[2:])
		// if is Unsupported item type of list, ignore this property
		if property.Items.Type == "undefined" {
			property = nil
			return
		}
	} else if strings.HasPrefix(typeAsString, "map[]") {
		property.Type = "map"
	} else if typeAsString == "time.Time" {
		property.Type = "Time"
	} else {
		property.Type = typeAsString
	}

	// fmt.Println(m.Id, field.Names, property.Type)

	if len(field.Names) == 0 {
		if astSelectorExpr, ok := field.Type.(*ast.SelectorExpr); ok {
			packageName := modelPackage
			if astTypeIdent, ok := astSelectorExpr.X.(*ast.Ident); ok {
				packageName = astTypeIdent.Name
			}
			name = packageName + "." + strings.TrimPrefix(astSelectorExpr.Sel.Name, "*")
		} else if astTypeIdent, ok := field.Type.(*ast.Ident); ok {
			name = astTypeIdent.Name
		} else if astStarExpr, ok := field.Type.(*ast.StarExpr); ok {
			if astIdent, ok := astStarExpr.X.(*ast.Ident); ok {
				name = astIdent.Name
			} else if astSelectorExpr, ok := astStarExpr.X.(*ast.SelectorExpr); ok {
				packageName := modelPackage
				if astTypeIdent, ok := astSelectorExpr.X.(*ast.Ident); ok {
					packageName = astTypeIdent.Name
				}

				name = packageName + "." + strings.TrimPrefix(astSelectorExpr.Sel.Name, "*")
			}
		} else {
			log.Fatalf("Something goes wrong: %#v", field.Type)
		}
		innerModel = &Model{}
		//log.Printf("Try to parse embeded type %s \n", name)
		//log.Fatalf("DEBUG: field: %#v\n, selector.X: %#v\n selector.Sel: %#v\n", field, astSelectorExpr.X, astSelectorExpr.Sel)
		knownModelNames := map[string]bool{}
		p.parseModel(innerModel, name, modelPackage, knownModelNames)

		for innerFieldName, innerField := range innerModel.Properties {
			_, exist := m.Properties[innerFieldName]
			if exist {
				continue
			}
			// fmt.Println("@@", m.Id, innerFieldName)
			m.Properties[innerFieldName] = innerField
		}

		//log.Fatalf("Here %#v\n", field.Type)
		return
	}
	name = field.Names[0].Name

	//log.Printf("ParseModelProperty: %s, CurrentPackage %s, type: %s \n", name, modelPackage, property.Type)
	//Analyse struct fields annotations
	if field.Tag != nil {
		structTag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		var tagText string
		if thriftTag := structTag.Get("thrift"); thriftTag != "" {
			tagText = thriftTag
		}
		if tag := structTag.Get("json"); tag != "" {
			tagText = tag
		}
		if tag := structTag.Get("example"); tag != "" {
			if strings.Contains(property.Type, "int") {
				property.Example, _ = strconv.Atoi(tag)
			} else if strings.Contains(property.Type, "float") {
				property.Example, _ = strconv.ParseFloat(tag, 64)
			} else if b, err := strconv.ParseBool(tag); err == nil {
				property.Example = b
			} else if property.Type == "array" {
				b, err := json.RawMessage(tag).MarshalJSON()
				if err != nil {
					property.Example = "invalid example"
				} else {
					sliceOfInterface := []interface{}{}
					err := json.Unmarshal(b, &sliceOfInterface)
					if err != nil {
						property.Example = "invalid example"
					} else {
						property.Example = sliceOfInterface
					}
				}
			} else if property.Type == "map" {
				b, err := json.RawMessage(tag).MarshalJSON()
				if err != nil {
					property.Example = "invalid example"
				} else {
					mapOfInterface := map[string]interface{}{}
					err := json.Unmarshal(b, &mapOfInterface)
					if err != nil {
						property.Example = "invalid example"
					} else {
						property.Example = mapOfInterface
					}
				}
			} else {
				property.Example = tag
			}
		}

		tagValues := strings.Split(tagText, ",")
		var isRequired = false

		for _, v := range tagValues {
			if v != "" && v != "required" && v != "omitempty" {
				name = v
			}
			if v == "required" {
				isRequired = true
			}
			// We will not document at all any fields with a json tag of "-"
			if v == "-" {
				return
			}
		}
		if required := structTag.Get("required"); required != "" || isRequired {
			m.Required = append(m.Required, name)
		}
		if desc := structTag.Get("description"); desc != "" {
			property.Description = desc
		}
	}
	m.Properties[name] = property
}

func (p *parser) setItemType(mp *ModelProperty, itemType string) {
	mp.Items = &ModelPropertyItems{}
	if isBasicType(itemType) {
		if isBasicTypeOASType(itemType) {
			mp.Items.Type = itemType
		} else {
			mp.Items.Type = "undefined"
		}
	} else {
		mp.Items.Ref = itemType
	}
}

func (p *parser) getPkgPathByPkgName(pkgName string) string {
	cachedPkgPath, ok := p.PkgPathCache[pkgName]
	if ok {
		return cachedPkgPath
	}
	return ""
}

func (p *parser) debug(v ...interface{}) {
	if p.Debug {
		log.Println(v...)
	}
}

func (p *parser) debugf(format string, args ...interface{}) {
	if p.Debug {
		log.Printf(format, args...)
	}
}

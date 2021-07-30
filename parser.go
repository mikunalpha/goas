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
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/iancoleman/orderedmap"
	module "golang.org/x/mod/modfile"
)

type parser struct {
	ModulePath string
	ModuleName string

	MainFilePath string

	HandlerPath string

	GoModFilePath string

	GoModCachePath string
	GoRootSrcPath  string

	OpenAPI        OpenAPIObject
	PackageAliases map[string]string

	CorePkgs      map[string]bool
	KnownPkgs     []pkg
	KnownNamePkg  map[string]*pkg
	KnownPathPkg  map[string]*pkg
	KnownIDSchema map[string]*SchemaObject

	TypeSpecs               map[string]map[string]*ast.TypeSpec
	PkgPathAstPkgCache      map[string]map[string]*ast.Package
	PkgNameImportedPkgAlias map[string]map[string][]string

	Debug bool
}

type pkg struct {
	Name string
	Path string
}

var (
	objectType = "object"
	stringType = "string"
	arrayType  = "array"
)

func newParser(modulePath, mainFilePath, handlerPath string, debug bool) (*parser, error) {
	p := &parser{
		CorePkgs:                map[string]bool{},
		KnownPkgs:               []pkg{},
		KnownNamePkg:            map[string]*pkg{},
		KnownPathPkg:            map[string]*pkg{},
		KnownIDSchema:           map[string]*SchemaObject{},
		TypeSpecs:               map[string]map[string]*ast.TypeSpec{},
		PkgPathAstPkgCache:      map[string]map[string]*ast.Package{},
		PkgNameImportedPkgAlias: map[string]map[string][]string{},
		Debug:                   debug,
	}
	p.OpenAPI.OpenAPI = OpenAPIVersion
	p.OpenAPI.Paths = make(PathsObject)
	p.OpenAPI.Security = []map[string][]string{}
	p.OpenAPI.Components.Schemas = make(map[string]*SchemaObject)
	p.OpenAPI.Components.SecuritySchemes = map[string]*SecuritySchemeObject{}

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
	// If GOPATH contains multiple paths use the last
	// TODO: choosing the last is arbritrary; handle this better
	goPathParts := strings.Split(goPath, ":")
	goPath = goPathParts[len(goPathParts)-1]
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
			return nil, fmt.Errorf("could not find goModCachePath: %w", err)
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", goModCachePath, err)
	}
	if !goModCacheInfo.IsDir() {
		return nil, fmt.Errorf("%s should be a directory", goModCachePath)
	}
	p.GoModCachePath = goModCachePath
	p.debugf("go module cache path: %s", p.GoModCachePath)

	goRoot := runtime.GOROOT()
	if goRoot == "" {
		return nil, fmt.Errorf("cannot get GOROOT")
	}
	goRootSrcPath := filepath.Join(goRoot, "src")
	_, err = os.Stat(goRootSrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get information of %s: %s", goRootSrcPath, err)
	}
	if !goModCacheInfo.IsDir() {
		return nil, fmt.Errorf("%s should be a directory", goRootSrcPath)
	}
	p.GoRootSrcPath = goRootSrcPath
	p.debugf("go root src path: %s", p.GoRootSrcPath)

	if handlerPath != "" {
		handlerPath, _ = filepath.Abs(handlerPath)
		_, err := os.Stat(handlerPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, err
			}
			return nil, fmt.Errorf("cannot get information of %s: %s", handlerPath, err)
		}
	}
	p.HandlerPath = handlerPath
	p.debugf("handler path: %s", p.HandlerPath)

	return p, nil
}

func (p *parser) parse() error {
	// parse basic info
	p.PackageAliases = make(map[string]string)

	err := p.parseEntryPoint()
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

	// parse core packages
	err = p.parseGoRoot()
	if err != nil {
		return err
	}

	// parse APIs info
	err = p.parseAPIs()
	if err != nil {
		return err
	}

	return nil
}

func (p *parser) CreateOASFile(path string) error {
	if err := p.parse(); err != nil {
		return err
	}

	fd, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Can not create the file %s: %v", path, err)
	}
	defer fd.Close()

	// for descriptions specified with $refs, pull that content in and embed it directly
	// TODO may be a good idea to make this optional via clarg
	err = p.explodeRefs()
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(p.OpenAPI, "", "  ")
	if err != nil {
		return err
	}
	_, err = fd.WriteString(string(output))

	return err
}

func (p *parser) explodeRefs() error {
	if p.OpenAPI.Info.Description != nil {
		desc, err := fetchRef(p.OpenAPI.Info.Description.Value)
		if err != nil {
			return err
		}
		p.OpenAPI.Info.Description.Value = desc
	}
	for i, tag := range p.OpenAPI.Tags {
		if tag.Description == nil {
			continue
		}
		desc, err := fetchRef(tag.Description.Value)
		if err != nil {
			return err
		}
		p.OpenAPI.Tags[i].Description.Value = desc
	}

	return nil
}

func fetchRef(description string) (string, error) {
	if !strings.HasPrefix(description, "$ref:") {
		return description, nil
	}
	url := description[5:]
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (p *parser) parseEntryPoint() error {
	fileTree, err := goparser.ParseFile(token.NewFileSet(), p.MainFilePath, nil, goparser.ParseComments)
	if err != nil {
		return fmt.Errorf("can not parse general API information: %v", err)
	}

	// Security Scopes are defined at a different level in the hierarchy as where they need to end up in the OpenAPI structure,
	// so a temporary list is needed.
	oauthScopes := make(map[string]map[string]string, 0)

	if fileTree.Comments != nil {
		for i := range fileTree.Comments {
			for _, comment := range strings.Split(fileTree.Comments[i].Text(), "\n") {
				attribute := strings.ToLower(strings.Split(comment, " ")[0])
				if len(attribute) == 0 || attribute[0] != '@' {
					continue
				}
				value := strings.TrimSpace(comment[len(attribute):])
				if len(value) == 0 {
					continue
				}
				// p.debug(attribute, value)
				switch attribute {
				case "@version":
					p.OpenAPI.Info.Version = value
				case "@title":
					p.OpenAPI.Info.Title = value
				case "@description":
					if p.OpenAPI.Info.Description == nil {
						p.OpenAPI.Info.Description = &ReffableString{}
					}
					p.OpenAPI.Info.Description.Value = value
				case "@termsofserviceurl":
					p.OpenAPI.Info.TermsOfService = value
				case "@contactname":
					if p.OpenAPI.Info.Contact == nil {
						p.OpenAPI.Info.Contact = &ContactObject{}
					}
					p.OpenAPI.Info.Contact.Name = value
				case "@contactemail":
					if p.OpenAPI.Info.Contact == nil {
						p.OpenAPI.Info.Contact = &ContactObject{}
					}
					p.OpenAPI.Info.Contact.Email = value
				case "@contacturl":
					if p.OpenAPI.Info.Contact == nil {
						p.OpenAPI.Info.Contact = &ContactObject{}
					}
					p.OpenAPI.Info.Contact.URL = value
				case "@licensename":
					if p.OpenAPI.Info.License == nil {
						p.OpenAPI.Info.License = &LicenseObject{}
					}
					p.OpenAPI.Info.License.Name = value
				case "@licenseurl":
					if p.OpenAPI.Info.License == nil {
						p.OpenAPI.Info.License = &LicenseObject{}
					}
					p.OpenAPI.Info.License.URL = value
				case "@server":
					fields := strings.Split(value, " ")
					s := ServerObject{URL: fields[0], Description: value[len(fields[0]):]}
					p.OpenAPI.Servers = append(p.OpenAPI.Servers, s)
				case "@security":
					fields := strings.Split(value, " ")
					security := map[string][]string{
						fields[0]: fields[1:],
					}
					p.OpenAPI.Security = append(p.OpenAPI.Security, security)
				case "@securityscheme":
					fields := strings.Split(value, " ")

					var scheme *SecuritySchemeObject
					if strings.Contains(fields[1], "oauth2") {
						if oauthScheme, ok := p.OpenAPI.Components.SecuritySchemes[fields[0]]; ok {
							scheme = oauthScheme
						} else {
							scheme = &SecuritySchemeObject{
								Type:       "oauth2",
								OAuthFlows: &SecuritySchemeOauthObject{},
							}
						}
					}

					if scheme == nil {
						scheme = &SecuritySchemeObject{
							Type: fields[1],
						}
					}
					switch fields[1] {
					case "http":
						scheme.Scheme = fields[2]
						scheme.Description = strings.Join(fields[3:], " ")
					case "apiKey":
						scheme.In = fields[2]
						scheme.Name = fields[3]
						scheme.Description = strings.Join(fields[4:], "")
					case "openIdConnect":
						scheme.OpenIdConnectUrl = fields[2]
						scheme.Description = strings.Join(fields[3:], " ")
					case "oauth2AuthCode":
						scheme.OAuthFlows.AuthorizationCode = &SecuritySchemeOauthFlowObject{
							AuthorizationUrl: fields[2],
							TokenUrl:         fields[3],
							Scopes:           make(map[string]string, 0),
						}
					case "oauth2Implicit":
						scheme.OAuthFlows.Implicit = &SecuritySchemeOauthFlowObject{
							AuthorizationUrl: fields[2],
							Scopes:           make(map[string]string, 0),
						}
					case "oauth2ResourceOwnerCredentials":
						scheme.OAuthFlows.ResourceOwnerPassword = &SecuritySchemeOauthFlowObject{
							TokenUrl: fields[2],
							Scopes:   make(map[string]string, 0),
						}
					case "oauth2ClientCredentials":
						scheme.OAuthFlows.ClientCredentials = &SecuritySchemeOauthFlowObject{
							TokenUrl: fields[2],
							Scopes:   make(map[string]string, 0),
						}
					}
					p.OpenAPI.Components.SecuritySchemes[fields[0]] = scheme
				case "@securityscope":
					fields := strings.Split(value, " ")

					if _, ok := oauthScopes[fields[0]]; !ok {
						oauthScopes[fields[0]] = make(map[string]string, 0)
					}

					oauthScopes[fields[0]][fields[1]] = strings.Join(fields[2:], " ")
				case "@tags":
					t, err := parseTags(comment)

					if err != nil {
						return err
					}

					p.OpenAPI.Tags = append(p.OpenAPI.Tags, *t)
				case "@packagealias":
					originalName, newName, err := parsePackageAliases(comment)

					if err != nil {
						return err
					}
					p.PackageAliases[originalName] = newName
				}
			}
		}
	}

	// Apply security scopes to their security schemes
	for scheme, _ := range p.OpenAPI.Components.SecuritySchemes {
		if p.OpenAPI.Components.SecuritySchemes[scheme].Type == "oauth2" {
			if scopes, ok := oauthScopes[scheme]; ok {
				p.OpenAPI.Components.SecuritySchemes[scheme].OAuthFlows.ApplyScopes(scopes)
			}
		}
	}

	if len(p.OpenAPI.Servers) < 1 {
		p.OpenAPI.Servers = append(p.OpenAPI.Servers, ServerObject{URL: "/", Description: "Default Server URL"})
	}

	if p.OpenAPI.Info.Title == "" {
		return fmt.Errorf("info.title cannot not be empty")
	}
	if p.OpenAPI.Info.Version == "" {
		return fmt.Errorf("info.version cannot not be empty")
	}
	for i := range p.OpenAPI.Servers {
		if p.OpenAPI.Servers[i].URL == "" {
			return fmt.Errorf("servers[%d].url cannot not be empty", i)
		}
	}

	return nil
}

func parsePackageAliases(comment string) (string, string, error) {
	re := regexp.MustCompile("\"([^\"]*)\"")
	matches := re.FindAllStringSubmatch(comment, -1)
	if len(matches) == 0 || len(matches[0]) == 1 {
		return "", "", fmt.Errorf("Expected: @PackageAlias \"<name>\" \"<alias>\"] Received: %s", comment)
	}

	return matches[0][1], matches[1][1], nil
}

func parseTags(comment string) (*TagDefinition, error) {
	re := regexp.MustCompile("\"([^\"]*)\"")
	matches := re.FindAllStringSubmatch(comment, -1)
	if len(matches) == 0 || len(matches[0]) == 1 {
		return nil, fmt.Errorf("Expected: @Tags \"<name>\" [\"<description>\"] Received: %s", comment)
	}
	tag := TagDefinition{Name: matches[0][1]}
	if len(matches) > 1 {
		tag.Description = &ReffableString{Value: matches[1][1]}
	}

	return &tag, nil
}

func (p *parser) parseModule() error {
	walker := func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			if strings.HasPrefix(strings.Trim(strings.TrimPrefix(path, p.ModulePath), "/"), ".git") {
				return nil
			}
			fns, err := filepath.Glob(filepath.Join(path, "*.go"))
			if len(fns) == 0 || err != nil {
				return nil
			}
			// p.debug(path)
			name := filepath.Join(p.ModuleName, strings.TrimPrefix(path, p.ModulePath))
			name = filepath.ToSlash(name)
			p.KnownPkgs = append(p.KnownPkgs, pkg{
				Name: name,
				Path: path,
			})
			p.KnownNamePkg[name] = &p.KnownPkgs[len(p.KnownPkgs)-1]
			p.KnownPathPkg[path] = &p.KnownPkgs[len(p.KnownPkgs)-1]
		}
		return nil
	}
	filepath.Walk(p.ModulePath, walker)
	return nil
}
func fixer(path, version string) (string, error) {
	return version, nil
}

func (p *parser) parseGoMod() error {
	b, err := ioutil.ReadFile(p.GoModFilePath)
	if err != nil {
		return err
	}
	goMod, err := module.ParseLax(p.GoModFilePath, b, fixer)
	if err != nil {
		return err
	}
	for i := range goMod.Require {
		pathRunes := []rune{}
		for _, v := range goMod.Require[i].Mod.Path {
			if !unicode.IsUpper(v) {
				pathRunes = append(pathRunes, v)
				continue
			}
			pathRunes = append(pathRunes, '!')
			pathRunes = append(pathRunes, unicode.ToLower(v))
		}
		pkgName := goMod.Require[i].Mod.Path
		pkgPath := filepath.Join(p.GoModCachePath, string(pathRunes)+"@"+goMod.Require[i].Mod.Version)
		pkgName = filepath.ToSlash(pkgName)
		p.KnownPkgs = append(p.KnownPkgs, pkg{
			Name: pkgName,
			Path: pkgPath,
		})
		p.KnownNamePkg[pkgName] = &p.KnownPkgs[len(p.KnownPkgs)-1]
		p.KnownPathPkg[pkgPath] = &p.KnownPkgs[len(p.KnownPkgs)-1]

		walker := func(path string, info os.FileInfo, err error) error {
			if info != nil && info.IsDir() {
				if strings.HasPrefix(strings.Trim(strings.TrimPrefix(path, p.ModulePath), "/"), ".git") {
					return nil
				}
				fns, err := filepath.Glob(filepath.Join(path, "*.go"))
				if len(fns) == 0 || err != nil {
					return nil
				}
				// p.debug(path)
				name := filepath.Join(pkgName, strings.TrimPrefix(path, pkgPath))
				name = filepath.ToSlash(name)
				p.KnownPkgs = append(p.KnownPkgs, pkg{
					Name: name,
					Path: path,
				})
				p.KnownNamePkg[name] = &p.KnownPkgs[len(p.KnownPkgs)-1]
				p.KnownPathPkg[path] = &p.KnownPkgs[len(p.KnownPkgs)-1]
			}
			return nil
		}
		filepath.Walk(pkgPath, walker)
	}
	if p.Debug {
		for i := range p.KnownPkgs {
			p.debug(p.KnownPkgs[i].Name, "->", p.KnownPkgs[i].Path)
		}
	}
	return nil
}

func (p *parser) parseGoRoot() error {
	walker := func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			fns, err := filepath.Glob(filepath.Join(path, "*.go"))
			if len(fns) == 0 || err != nil {
				return nil
			}
			name := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(path, p.GoRootSrcPath)), "/")
			p.CorePkgs[name] = true
		}
		return nil
	}
	filepath.Walk(p.GoRootSrcPath, walker)
	return nil
}

func (p *parser) getPkgAst(pkgPath string) (map[string]*ast.Package, error) {
	if cache, ok := p.PkgPathAstPkgCache[pkgPath]; ok {
		return cache, nil
	}
	ignoreFileFilter := func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}
	astPackages, err := goparser.ParseDir(token.NewFileSet(), pkgPath, ignoreFileFilter, goparser.ParseComments)
	if err != nil {
		return nil, err
	}
	p.PkgPathAstPkgCache[pkgPath] = astPackages
	return astPackages, nil
}

func (p *parser) parseAPIs() error {
	err := p.parseImportStatements()
	if err != nil {
		return err
	}

	err = p.parseTypeSpecs()
	if err != nil {
		return err
	}

	return p.parsePaths()
}

func (p *parser) parseImportStatements() error {
	for i := range p.KnownPkgs {
		pkgPath := p.KnownPkgs[i].Path
		pkgName := p.KnownPkgs[i].Name

		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			p.debugf("parseImportStatements: parse of %s package cause error: %s\n", pkgPath, err)
			continue
		}

		p.PkgNameImportedPkgAlias[pkgName] = map[string][]string{}
		for _, astPackageKey := range sortedPackageKeys(astPkgs) {
			astPackage := astPkgs[astPackageKey]
			for _, astFileKey := range sortedFileKeys(astPackage.Files) {
				astFile := astPackage.Files[astFileKey]
				for _, astImport := range astFile.Imports {
					importedPkgName := strings.Trim(astImport.Path.Value, "\"")
					importedPkgAlias := ""

					// _, known := p.KnownNamePkg[importedPkgName]
					// if !known {
					// 	p.debug("unknown", importedPkgName)
					// }

					if astImport.Name != nil && astImport.Name.Name != "." && astImport.Name.Name != "_" {
						importedPkgAlias = astImport.Name.String()
						// p.debug(importedPkgAlias, importedPkgName)
					} else {
						s := strings.Split(importedPkgName, "/")
						importedPkgAlias = s[len(s)-1]
					}

					exist := false
					for _, v := range p.PkgNameImportedPkgAlias[pkgName][importedPkgAlias] {
						if v == importedPkgName {
							exist = true
							break
						}
					}
					if !exist {
						p.PkgNameImportedPkgAlias[pkgName][importedPkgAlias] = append(p.PkgNameImportedPkgAlias[pkgName][importedPkgAlias], importedPkgName)
					}
				}
			}
		}
	}
	return nil
}

func (p *parser) parseTypeSpecs() error {
	for i := range p.KnownPkgs {
		pkgPath := p.KnownPkgs[i].Path
		pkgName := p.KnownPkgs[i].Name

		_, ok := p.TypeSpecs[pkgName]
		if !ok {
			p.TypeSpecs[pkgName] = map[string]*ast.TypeSpec{}
		}
		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			p.debugf("parseTypeSpecs: parse of %s package cause error: %s\n", pkgPath, err)
			continue
		}
		for _, astPackageKey := range sortedPackageKeys(astPkgs) {
			astPackage := astPkgs[astPackageKey]
			for _, astFileKey := range sortedFileKeys(astPackage.Files) {
				astFile := astPackage.Files[astFileKey]
				for _, astDeclaration := range astFile.Decls {
					if astGenDeclaration, ok := astDeclaration.(*ast.GenDecl); ok && astGenDeclaration.Tok == token.TYPE {
						// find type declaration
						for _, astSpec := range astGenDeclaration.Specs {
							if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
								p.TypeSpecs[pkgName][typeSpec.Name.String()] = typeSpec
							}
						}
					} else if astFuncDeclaration, ok := astDeclaration.(*ast.FuncDecl); ok {
						// find type declaration in func, method
						if astFuncDeclaration.Doc != nil && astFuncDeclaration.Doc.List != nil && astFuncDeclaration.Body != nil {
							funcName := astFuncDeclaration.Name.String()
							for _, astStmt := range astFuncDeclaration.Body.List {
								if astDeclStmt, ok := astStmt.(*ast.DeclStmt); ok {
									if astGenDeclaration, ok := astDeclStmt.Decl.(*ast.GenDecl); ok {
										for _, astSpec := range astGenDeclaration.Specs {
											if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
												// type in func
												if astFuncDeclaration.Recv == nil {
													p.TypeSpecs[pkgName][strings.Join([]string{funcName, typeSpec.Name.String()}, "@")] = typeSpec
													continue
												}
												// type in method
												var recvTypeName string
												if astStarExpr, ok := astFuncDeclaration.Recv.List[0].Type.(*ast.StarExpr); ok {
													recvTypeName = fmt.Sprintf("%s", astStarExpr.X)
												} else if astIdent, ok := astFuncDeclaration.Recv.List[0].Type.(*ast.Ident); ok {
													recvTypeName = astIdent.String()
												}
												p.TypeSpecs[pkgName][strings.Join([]string{recvTypeName, funcName, typeSpec.Name.String()}, "@")] = typeSpec
											}
										}
									}
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

func (p *parser) parsePaths() error {
	for i := range p.KnownPkgs {
		pkgPath := p.KnownPkgs[i].Path
		pkgName := p.KnownPkgs[i].Name
		// p.debug(pkgName, "->", pkgPath)

		astPkgs, err := p.getPkgAst(pkgPath)
		if err != nil {
			p.debugf("parsePaths: parse of %s package cause error: %s\n", pkgPath, err)
			continue
		}
		for _, astPackageKey := range sortedPackageKeys(astPkgs) {
			astPackage := astPkgs[astPackageKey]
			for _, astFileKey := range sortedFileKeys(astPackage.Files) {
				astFile := astPackage.Files[astFileKey]
				for _, astDeclaration := range astFile.Decls {
					if astFuncDeclaration, ok := astDeclaration.(*ast.FuncDecl); ok {
						if astFuncDeclaration.Doc != nil && astFuncDeclaration.Doc.List != nil {
							err = p.parseOperation(pkgPath, pkgName, astFuncDeclaration.Doc.List)
							if err != nil {
								return err
							}
						}
					} else if astVarDeclaration, ok := astDeclaration.(*ast.GenDecl); ok {
						if astVarDeclaration.Doc != nil && astVarDeclaration.Doc.List != nil {
							err = p.parseOperation(pkgPath, pkgName, astVarDeclaration.Doc.List)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func isHidden(astComments []*ast.Comment) bool {
	for _, astComment := range astComments {
		comment := strings.TrimSpace(strings.TrimLeft(astComment.Text, "/"))
		if len(comment) == 0 {
			// ignore empty lines
			continue
		}
		attribute := strings.Fields(comment)[0]
		if strings.ToLower(attribute) == "@hidden" {
			return true
		}
	}
	return false
}

func (p *parser) parseOperation(pkgPath, pkgName string, astComments []*ast.Comment) error {
	operation := &OperationObject{
		Responses: map[string]*ResponseObject{},
	}
	if !strings.HasPrefix(pkgPath, p.ModulePath) {
		// ignore this pkgName
		// p.debugf("parseOperation ignores %s", pkgPath)
		return nil
	} else if p.HandlerPath != "" && !strings.HasPrefix(pkgPath, p.HandlerPath) {
		return nil
	}
	if isHidden(astComments) {
		return nil
	}
	var err error
	for _, astComment := range astComments {
		comment := strings.TrimSpace(strings.TrimLeft(astComment.Text, "/"))
		if len(comment) == 0 {
			// ignore empty lines
			continue
		}
		attribute := strings.Fields(comment)[0]
		value := strings.TrimSpace(comment[len(attribute):])
		switch strings.ToLower(attribute) {
		case "@title":
			operation.Summary = value
		case "@description":
			err = p.parseDescription(operation, value)
		case "@operationid":
			operation.OperationID = value
		case "@param":
			err = p.parseParamComment(pkgPath, pkgName, operation, value)
		case "@success", "@failure":
			err = p.parseResponseComment(pkgPath, pkgName, operation, value)
		case "@resource", "@tag":
			resource := value
			if resource == "" {
				resource = "others"
			}
			if !isInStringList(operation.Tags, resource) {
				operation.Tags = append(operation.Tags, resource)
			}
		case "@route", "@router":
			err = p.parseRouteComment(operation, comment)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) parseDescription(operation *OperationObject, description string) error {
	desc, err := fetchRef(description)
	if err != nil {
		return err
	}
	operation.Description = strings.Join([]string{operation.Description, desc}, " ")
	return nil
}

func (p *parser) parseParamComment(pkgPath, pkgName string, operation *OperationObject, comment string) error {
	// {name}  {in}  {goType}  {required}  {description}  		{example (optional)}
	// user    body  User      true        "Info of a user."	"{\"name\":\"Bilbo\"}"
	// f       file  ignored   true        "Upload a file."
	re := regexp.MustCompile(`([-\w]+)[\s]+([\w]+)[\s]+([\w./\[\]\\(\\),]+)[\s]+([\w]+)[\s]+"([^"]+)"(?:[\s]+"((?:[^"\\]|\\")*)")?`)
	matches := re.FindStringSubmatch(comment)
	if len(matches) < 6 {
		return fmt.Errorf("parseParamComment can not parse param comment \"%s\"", comment)
	}
	name := matches[1]
	in := matches[2]

	re = regexp.MustCompile(`\[\w*\]`)
	goType := re.ReplaceAllString(matches[3], "[]")

	required := false
	switch strings.ToLower(matches[4]) {
	case "true", "required":
		required = true
	}
	description := matches[5]

	// `file`, `form`
	if in == "file" || in == "files" || in == "form" {
		if operation.RequestBody == nil {
			operation.RequestBody = &RequestBodyObject{
				Content: map[string]*MediaTypeObject{
					ContentTypeForm: &MediaTypeObject{
						Schema: SchemaObject{
							Type:       &objectType,
							Properties: orderedmap.New(),
						},
					},
				},
				Required: required,
			}
		}
		if in == "file" {
			operation.RequestBody.Content[ContentTypeForm].Schema.Properties.Set(name, &SchemaObject{
				Type:        &stringType,
				Format:      "binary",
				Description: description,
			})
		} else if in == "files" {
			operation.RequestBody.Content[ContentTypeForm].Schema.Properties.Set(name, &SchemaObject{
				Type: &arrayType,
				Items: &SchemaObject{
					Type:   &stringType,
					Format: "binary",
				},
				Description: description,
			})
		} else if isGoTypeOASType(goType) {
			localGoType := goTypesOASTypes[goType]
			operation.RequestBody.Content[ContentTypeForm].Schema.Properties.Set(name, &SchemaObject{
				Type:        &localGoType,
				Format:      goTypesOASFormats[goType],
				Description: description,
			})
		}
		return nil
	}

	// `path`, `query`, `header`, `cookie`
	if in != "body" {
		parameterObject := ParameterObject{
			Name:        name,
			In:          in,
			Description: description,
			Required:    required,
		}
		if in == "path" {
			parameterObject.Required = true
		}
		if goType == "time.Time" {
			var err error
			parameterObject.Schema, err = p.parseSchemaObject(pkgPath, pkgName, goType, true)
			if err != nil {
				p.debug("parseResponseComment cannot parse goType", goType)
			}
			operation.Parameters = append(operation.Parameters, parameterObject)
		} else if isGoTypeOASType(goType) {
			localGoType := goTypesOASTypes[goType]
			parameterObject.Schema = &SchemaObject{
				Type:        &localGoType,
				Format:      goTypesOASFormats[goType],
				Description: description,
			}
			operation.Parameters = append(operation.Parameters, parameterObject)
		}
		return nil
	}

	if operation.RequestBody == nil {
		operation.RequestBody = &RequestBodyObject{
			Content:  map[string]*MediaTypeObject{},
			Required: required,
		}
	}

	s, err := p.parseBodyType(pkgPath, pkgName, goType)
	if err != nil {
		return err
	}
	operation.RequestBody.Content[ContentTypeJson] = &MediaTypeObject{
		Schema: *s,
	}
	// parse example
	if len(matches) > 6 && matches[6] != "" {
		exampleRequestBody, err := parseRequestBodyExample(matches[6])
		if err != nil {
			return err
		}
		operation.RequestBody.Content[ContentTypeJson].Example = exampleRequestBody
	}

	return nil
}

func parseRequestBodyExample(example string) (interface{}, error) {
	exampleRequestBody := map[string]interface{}{}
	err := json.Unmarshal([]byte(strings.Replace(example, "\\\"", "\"", -1)), &exampleRequestBody)
	if err != nil {
		return nil, err
	}
	return exampleRequestBody, nil
}

func (p *parser) parseBodyType(pkgPath, pkgName, typeName string) (*SchemaObject, error) {
	if strings.HasPrefix(typeName, "[]") || strings.HasPrefix(typeName, "map[]") || typeName == "time.Time" {
		schema, err := p.parseSchemaObject(pkgPath, pkgName, typeName, true)
		if err != nil {
			p.debug("parseResponseComment cannot parse type", typeName)
		}
		return schema, nil
	}

	// handle oneOf/anyOf/allOf/not
	sob, err := p.handleCompoundType(pkgPath, pkgName, typeName)
	if sob != nil || err != nil {
		return sob, err
	}

	registeredTypeName, err := p.registerType(pkgPath, pkgName, typeName)
	if err != nil {
		return nil, err
	}
	if isBasicGoType(registeredTypeName) {
		return &SchemaObject{
			Type: &stringType,
		}, nil
	} else {
		return &SchemaObject{
			Ref: addSchemaRefLinkPrefix(registeredTypeName),
		}, nil
	}
}

func (p *parser) parseResponseComment(pkgPath, pkgName string, operation *OperationObject, comment string) error {
	// {status}  {jsonType}  {goType}     {description}
	// 201       object      models.User  "User Model"
	// if 204 or something else without empty return payload
	// 204 "User Model"
	re := regexp.MustCompile(`(?P<status>[\d]+)[\s]*(?P<jsonType>[\w\{\}]+)?[\s]+(?P<goType>[\w\-\.\/\[\]]+)?[^"]*(?P<description>.*)?`)
	matches := re.FindStringSubmatch(comment)

	paramsMap := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i > 0 && i <= len(matches) {
			paramsMap[name] = matches[i]
		}
	}

	if len(matches) <= 2 {
		return fmt.Errorf("parseResponseComment can not parse response comment \"%s\", matches: %v", comment, matches)
	}

	status := paramsMap["status"]
	_, err := strconv.Atoi(status)
	if err != nil {
		return fmt.Errorf("parseResponseComment: http status must be int, but got %s", status)
	}

	// ignore type if not set
	if jsonType := paramsMap["jsonType"]; jsonType != "" {
		switch jsonType {
		case "object", "array", "{object}", "{array}":
		default:
			return fmt.Errorf("parseResponseComment: invalid jsonType \"%s\"", paramsMap["jsonType"])
		}
	}

	responseObject := &ResponseObject{
		Content: map[string]*MediaTypeObject{},
	}
	responseObject.Description = strings.Trim(paramsMap["description"], "\"")

	if goTypeRaw := paramsMap["goType"]; goTypeRaw != "" {
		re = regexp.MustCompile(`\[\w*\]`)
		goType := re.ReplaceAllString(goTypeRaw, "[]")
		if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[]") {
			schema, err := p.parseSchemaObject(pkgPath, pkgName, goType, true)
			if err != nil {
				p.debug("parseResponseComment: cannot parse goType", goType)
			}
			responseObject.Content[ContentTypeJson] = &MediaTypeObject{
				Schema: *schema,
			}
		} else {
			typeName, err := p.registerType(pkgPath, pkgName, matches[3])
			if err != nil {
				return err
			}
			if isBasicGoType(typeName) {
				responseObject.Content[ContentTypeText] = &MediaTypeObject{
					Schema: SchemaObject{
						Type: &stringType,
					},
				}
			} else {
				responseObject.Content[ContentTypeJson] = &MediaTypeObject{
					Schema: SchemaObject{
						Ref: addSchemaRefLinkPrefix(typeName),
					},
				}
			}
		}
	}
	operation.Responses[status] = responseObject

	return nil
}

func (p *parser) routeAndMethodExist(route string, method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return p.OpenAPI.Paths[route].Get != nil
	case http.MethodPost:
		return p.OpenAPI.Paths[route].Post != nil
	case http.MethodPatch:
		return p.OpenAPI.Paths[route].Patch != nil
	case http.MethodPut:
		return p.OpenAPI.Paths[route].Put != nil
	case http.MethodDelete:
		return p.OpenAPI.Paths[route].Delete != nil
	case http.MethodOptions:
		return p.OpenAPI.Paths[route].Options != nil
	case http.MethodHead:
		return p.OpenAPI.Paths[route].Head != nil
	case http.MethodTrace:
		return p.OpenAPI.Paths[route].Trace != nil
	}

	return false
}

func (p *parser) parseRouteComment(operation *OperationObject, comment string) error {
	sourceString := strings.TrimSpace(comment[len("@Router"):])

	// /path [method]
	re := regexp.MustCompile(`([\w\.\/\-{}]+)[^\[]+\[([^\]]+)`)
	matches := re.FindStringSubmatch(sourceString)
	if len(matches) != 3 {
		return fmt.Errorf("Can not parse router comment \"%s\", skipped", comment)
	}

	_, ok := p.OpenAPI.Paths[matches[1]]
	if !ok {
		p.OpenAPI.Paths[matches[1]] = &PathItemObject{}
	} else if p.routeAndMethodExist(matches[1], matches[2]) {
		return fmt.Errorf("Already exists, %q [%q]", matches[1], matches[2])
	}

	switch strings.ToUpper(matches[2]) {
	case http.MethodGet:
		p.OpenAPI.Paths[matches[1]].Get = operation
	case http.MethodPost:
		p.OpenAPI.Paths[matches[1]].Post = operation
	case http.MethodPatch:
		p.OpenAPI.Paths[matches[1]].Patch = operation
	case http.MethodPut:
		p.OpenAPI.Paths[matches[1]].Put = operation
	case http.MethodDelete:
		p.OpenAPI.Paths[matches[1]].Delete = operation
	case http.MethodOptions:
		p.OpenAPI.Paths[matches[1]].Options = operation
	case http.MethodHead:
		p.OpenAPI.Paths[matches[1]].Head = operation
	case http.MethodTrace:
		p.OpenAPI.Paths[matches[1]].Trace = operation
	}

	return nil
}

func (p *parser) getSchemaObjectCached(pkgPath, pkgName, typeName string) (*SchemaObject, error) {
	var schemaObject *SchemaObject

	// see if we've already parsed this type
	cachedObj := p.checkCache(pkgName, typeName)
	if cachedObj != nil {
		return cachedObj, nil
	} else {
		// if not, parse it now
		parsedObject, err := p.parseSchemaObject(pkgPath, pkgName, typeName, true)
		if err != nil {
			return schemaObject, err
		}
		schemaObject = parsedObject
	}

	return schemaObject, nil
}

func (p *parser) registerType(pkgPath, pkgName, typeName string) (string, error) {
	var registerTypeName string

	if isBasicGoType(typeName) {
		registerTypeName = typeName
	} else {
		schemaObject, err := p.getSchemaObjectCached(pkgPath, pkgName, typeName)
		if err != nil {
			return "", err
		}
		registerTypeName = schemaObject.ID
	}

	if registerTypeName == "" {
		return "", errors.New(fmt.Sprintf("Could not parse schema for %s %s %s", pkgName, pkgName, typeName))
	}

	return registerTypeName, nil
}

func trimSplit(csl string) []string {
	s := strings.Split(csl, ",")
	for i := range s {
		s[i] = strings.TrimSpace(s[i])
	}
	return s
}

func (p *parser) handleCompoundType(pkgPath, pkgName, typeName string) (*SchemaObject, error) {
	re := regexp.MustCompile("(?i)(oneOf|anyOf|allOf|not)\\(([^\\)]*)\\)")
	matches := re.FindStringSubmatch(typeName)
	if len(matches) < 3 {
		return nil, nil
	}
	op := strings.ToLower(matches[1])
	if matches[2] == "" {
		return nil, fmt.Errorf("Expected 1 or more arguments, received '%s'", typeName)
	}
	args := trimSplit(matches[2])

	// not only supports one arg
	if op == "not" && len(args) != 1 {
		return nil, fmt.Errorf("Invalid number of arguments for not compound type, expected 1 received %d", len(args))
	}

	var sobs []*SchemaObject
	for i := range args {
		result, err := p.parseBodyType(pkgPath, pkgName, args[i])
		if err != nil {
			return nil, err
		}
		sobs = append(sobs, result)
	}

	sob := &SchemaObject{}
	switch op {
	case "not":
		sob.Not = sobs[0]
	case "oneof":
		sob.OneOf = sobs
	case "anyof":
		sob.AnyOf = sobs
	case "allof":
		sob.AllOf = sobs
	default:
		return nil, fmt.Errorf("Invalid compound type '%s'", op)
	}

	return sob, nil
}

func (p *parser) parseSchemaObject(pkgPath, pkgName, typeName string, register bool) (*SchemaObject, error) {
	var typeSpec *ast.TypeSpec
	var exist bool
	var schemaObject SchemaObject
	var err error

	// handler basic and some specific typeName
	if strings.HasPrefix(typeName, "[]") {
		schemaObject.Type = &arrayType
		itemTypeName := typeName[2:]
		schema, ok := p.KnownIDSchema[genSchemaObjectID(pkgName, itemTypeName, p.PackageAliases)]
		if ok {
			schemaObject.Items = &SchemaObject{Ref: addSchemaRefLinkPrefix(schema.ID)}
			return &schemaObject, nil
		}
		schemaObject.Items, err = p.parseSchemaObject(pkgPath, pkgName, itemTypeName, true)
		if err != nil {
			return nil, err
		}
		return &schemaObject, nil
	} else if strings.HasPrefix(typeName, "map[]") {
		schemaObject.Type = &objectType
		itemTypeName := typeName[5:]
		schema, ok := p.KnownIDSchema[genSchemaObjectID(pkgName, itemTypeName, p.PackageAliases)]
		if ok {
			schemaObject.AdditionalProperties = &SchemaObject{Ref: addSchemaRefLinkPrefix(schema.ID)}
			return &schemaObject, nil
		}
		schemaProperty, err := p.parseSchemaObject(pkgPath, pkgName, itemTypeName, true)
		if err != nil {
			return nil, err
		}
		schemaObject.AdditionalProperties = schemaProperty
		return &schemaObject, nil
	} else if typeName == "time.Time" {
		schemaObject.Type = &stringType
		schemaObject.Format = "date-time"
		return &schemaObject, nil
	} else if strings.HasPrefix(typeName, "interface{}") {
		schemaObject.Type = nil
		return &schemaObject, nil
	} else if isGoTypeOASType(typeName) {
		localGoType := goTypesOASTypes[typeName]
		schemaObject.Type = &localGoType
		return &schemaObject, nil
	}

	// handler other type
	typeNameParts := strings.Split(typeName, ".")
	if len(typeNameParts) == 1 {
		typeSpec, exist = p.getTypeSpec(pkgPath, pkgName, typeName)
		if !exist {
			for _, value := range p.KnownNamePkg {
				typeSpec, exist = p.getTypeSpec(value.Path, value.Name, typeName)
				if exist {
					pkgPath = value.Path
					pkgName = value.Name
					break
				}
			}
			if !exist {
				log.Fatalf("Can not find definition of %s ast.TypeSpec. Current package %s", typeName, pkgName)
			}
		}
		schemaObject.PkgName = pkgName
		schemaObject.ID = genSchemaObjectID(pkgName, typeName, p.PackageAliases)
		p.KnownIDSchema[schemaObject.ID] = &schemaObject
	} else {
		guessPkgName := strings.Join(typeNameParts[:len(typeNameParts)-1], "/")
		guessPkgPath := ""
		for i := range p.KnownPkgs {
			if guessPkgName == p.KnownPkgs[i].Name {
				guessPkgPath = p.KnownPkgs[i].Path
				break
			}
		}
		guessTypeName := typeNameParts[len(typeNameParts)-1]
		typeSpec, exist = p.getTypeSpec(guessPkgName, guessPkgName, guessTypeName)
		if !exist {
			found := false
			for k := range p.PkgNameImportedPkgAlias[pkgName] {
				if k == guessPkgName && len(p.PkgNameImportedPkgAlias[pkgName][guessPkgName]) != 0 {
					found = true
					break
				}
			}
			if !found {
				p.debugf("unknown guess %s ast.TypeSpec in package %s", guessTypeName, guessPkgName)
				return &schemaObject, nil
			}
			guessPkgName = p.PkgNameImportedPkgAlias[pkgName][guessPkgName][0]
			guessPkgPath = ""
			for i := range p.KnownPkgs {
				if guessPkgName == p.KnownPkgs[i].Name {
					guessPkgPath = p.KnownPkgs[i].Path
					break
				}
			}
			// p.debugf("guess %s ast.TypeSpec in package %s", guessTypeName, guessPkgName)

			typeSpec, exist = p.getTypeSpec(guessPkgPath, guessPkgName, guessTypeName)
			if !exist {
				if p.CorePkgs[guessPkgName] == true {
					p.debugf("Ignoring missing type %s in core package %s", guessTypeName, guessPkgName)
					schemaObject.Type = &objectType
					return &schemaObject, nil
				}

				log.Fatalf("Cannot find definition of guess %s ast.TypeSpec in package %s. "+
					"If definition is in a vendor dependency, try running `go mod tidy && go mod vendor`",
					guessTypeName, guessPkgName)
			}
			schemaObject.PkgName = guessPkgName
			schemaObject.ID = genSchemaObjectID(guessPkgName, guessTypeName, p.PackageAliases)
			p.KnownIDSchema[schemaObject.ID] = &schemaObject
		}
		pkgPath, pkgName = guessPkgPath, guessPkgName
	}

	if isGoTypeOASType(p.getTypeAsString(typeSpec.Type)) && schemaObject.Ref == "" {
		typeAsString := p.getTypeAsString(typeSpec.Type)
		localGoType := goTypesOASTypes[typeAsString]
		schemaObject.Type = &localGoType
		checkFormatInt64(typeAsString, &schemaObject)

	} else if astIdent, ok := typeSpec.Type.(*ast.Ident); ok {
		// this is for type aliases to custom types
		newSchema, err := p.parseSchemaObject(pkgPath, pkgName, astIdent.Name, true)
		if err != nil {
			return nil, err
		}
		schemaObject.Type = newSchema.Type
		schemaObject.Properties = newSchema.Properties
		schemaObject.AdditionalProperties = newSchema.AdditionalProperties
	} else if astStructType, ok := typeSpec.Type.(*ast.StructType); ok {
		schemaObject.Type = &objectType
		if astStructType.Fields != nil {
			p.parseSchemaPropertiesFromStructFields(pkgPath, pkgName, &schemaObject, astStructType.Fields.List)
		}
	} else if astArrayType, ok := typeSpec.Type.(*ast.ArrayType); ok {
		schemaObject.Type = &arrayType
		schemaObject.Items = &SchemaObject{}
		typeAsString := p.getTypeAsString(astArrayType.Elt)
		typeAsString = strings.TrimLeft(typeAsString, "*")

		if !isBasicGoType(typeAsString) {
			itemsSchema, err := p.getSchemaObjectCached(pkgPath, pkgName, typeAsString)
			if err != nil {
				p.debug("parseSchemaObject parse array items err:", err)
			} else {
				if itemsSchema.ID != "" {
					schemaObject.Items.Ref = addSchemaRefLinkPrefix(itemsSchema.ID)
				} else {
					*schemaObject.Items = *itemsSchema
				}
			}
		} else if isGoTypeOASType(typeAsString) {
			localGoType := goTypesOASTypes[typeAsString]
			schemaObject.Items.Type = &localGoType
		}
	} else if astMapType, ok := typeSpec.Type.(*ast.MapType); ok {
		schemaObject.Type = &objectType
		propertySchema := &SchemaObject{}
		schemaObject.AdditionalProperties = propertySchema
		typeAsString := p.getTypeAsString(astMapType.Value)
		typeAsString = strings.TrimLeft(typeAsString, "*")
		if !isBasicGoType(typeAsString) {
			keySchema, err := p.getSchemaObjectCached(pkgPath, pkgName, typeAsString)
			if err != nil {
				p.debug("parseSchemaObject parse array items err:", err)
			} else {
				if keySchema.ID != "" {
					propertySchema.Ref = addSchemaRefLinkPrefix(keySchema.ID)
				} else {
					*propertySchema = *keySchema
				}
			}
		} else if isGoTypeOASType(typeAsString) {
			localGoType := goTypesOASTypes[typeAsString]
			propertySchema.Type = &localGoType
		}
	} else if selectorType, ok := typeSpec.Type.(*ast.SelectorExpr); ok {
		// this case is for referencing third party packages.
		packageIdentifier, ok := selectorType.X.(*ast.Ident)
		usedTypeName := selectorType.Sel.Name
		if ok {
			packageName := packageIdentifier.Name
			for potentialPackage, typeSpecs := range p.TypeSpecs {
				if strings.HasSuffix(potentialPackage, packageName) {
					// iterate through types of that package
					for name, _ := range typeSpecs {
						if name == usedTypeName {
							parsedPackageSchema, err := p.parseSchemaObject(potentialPackage, potentialPackage, usedTypeName, false)
							if err != nil {
								return nil, err
							}
							schemaObject.Type = parsedPackageSchema.Type
							schemaObject.Properties = parsedPackageSchema.Properties
							schemaObject.AdditionalProperties = parsedPackageSchema.AdditionalProperties
							break
						}
					}
				}
			}

		}
	} else if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
		// free form object since the interface can be "anything"
		schemaObject.Type = nil
	}

	// we don't want to register 3rd party library types
	if register {
		// register schema object in spec tree if it doesn't exist
		registerTypeName := schemaObject.ID
		_, ok := p.OpenAPI.Components.Schemas[replaceBackslash(registerTypeName)]
		if !ok {
			p.OpenAPI.Components.Schemas[replaceBackslash(registerTypeName)] = &schemaObject
		}
	}

	return &schemaObject, nil
}

func (p *parser) getTypeSpec(pkgPath, pkgName, typeName string) (*ast.TypeSpec, bool) {
	pkgTypeSpecs, exist := p.TypeSpecs[pkgName]
	if !exist {
		return nil, false
	}
	astTypeSpec, exist := pkgTypeSpecs[typeName]
	if !exist {
		return nil, false
	}
	return astTypeSpec, true
}

func (p *parser) parseSchemaPropertiesFromStructFields(pkgPath, pkgName string, structSchema *SchemaObject, astFields []*ast.Field) {
	if astFields == nil {
		return
	}
	structSchema.Properties = orderedmap.New()
	if structSchema.DisabledFieldNames == nil {
		structSchema.DisabledFieldNames = map[string]struct{}{}
	}

	for _, astField := range astFields {
		fieldSchema := &SchemaObject{}
		typeAsString := p.getTypeAsString(astField.Type)
		typeAsString = strings.TrimLeft(typeAsString, "*")
		isSliceOrMap := strings.HasPrefix(typeAsString, "[]") || strings.HasPrefix(typeAsString, "map[]")
		isInterface := strings.HasPrefix(typeAsString, "interface{}")
		if isSliceOrMap || isInterface || typeAsString == "time.Time" {
			var err error
			fieldSchema, err = p.parseSchemaObject(pkgPath, pkgName, typeAsString, true)
			if err != nil {
				p.debug(err)
				return
			}
		} else if !isBasicGoType(typeAsString) {
			fieldSchemaSchemeaObjectID, err := p.registerType(pkgPath, pkgName, typeAsString)
			if err != nil {
				p.debug("parseSchemaPropertiesFromStructFields err:", err)
			} else {
				fieldSchema.ID = fieldSchemaSchemeaObjectID
				_, ok := p.KnownIDSchema[fieldSchemaSchemeaObjectID]
				if ok {
					fieldSchema.Ref = addSchemaRefLinkPrefix(fieldSchemaSchemeaObjectID)
				} else {
					fieldSchema, err = p.parseSchemaObject(pkgPath, pkgName, typeAsString, true)
					if err != nil {
						p.debug(err)
						return
					}
				}
			}
		} else if isGoTypeOASType(typeAsString) {
			localGoType := goTypesOASTypes[typeAsString]
			fieldSchema.Type = &localGoType
			checkFormatInt64(typeAsString, fieldSchema)
		}
		// for embedded fields
		if len(astField.Names) == 0 {
			if fieldSchema.Properties != nil {
				for _, propertyName := range fieldSchema.Properties.Keys() {
					_, exist := structSchema.Properties.Get(propertyName)
					if exist {
						continue
					}
					propertySchema, _ := fieldSchema.Properties.Get(propertyName)
					structSchema.Properties.Set(propertyName, propertySchema)
				}
			} else if len(fieldSchema.Ref) != 0 && len(fieldSchema.ID) != 0 {
				refSchema, ok := p.KnownIDSchema[fieldSchema.ID]
				if ok {
					if refSchema.Properties == nil {
						p.debug("nil refSchema.Properties")
						continue
					}
					for _, propertyName := range refSchema.Properties.Keys() {
						refPropertySchema, _ := refSchema.Properties.Get(propertyName)
						_, disabled := structSchema.DisabledFieldNames[refPropertySchema.(*SchemaObject).FieldName]
						if disabled {
							continue
						}
						_, exist := structSchema.Properties.Get(propertyName)
						if exist {
							continue
						}

						structSchema.Properties.Set(propertyName, refPropertySchema)
					}
				}
			}
		} else {
			name := astField.Names[0].Name
			fieldSchema.FieldName = name
			_, disabled := structSchema.DisabledFieldNames[name]
			if disabled {
				continue
			}

			newName, skip := parseStructTags(astField, structSchema, fieldSchema, name)
			if skip {
				continue
			}

			name = newName

			structSchema.Properties.Set(name, fieldSchema)
		}
	}
}

func (p *parser) getTypeAsString(fieldType interface{}) string {
	astArrayType, ok := fieldType.(*ast.ArrayType)
	if ok {
		return fmt.Sprintf("[]%v", p.getTypeAsString(astArrayType.Elt))
	}

	astMapType, ok := fieldType.(*ast.MapType)
	if ok {
		return fmt.Sprintf("map[]%v", p.getTypeAsString(astMapType.Value))
	}

	_, ok = fieldType.(*ast.InterfaceType)
	if ok {
		return "interface{}"
	}

	astStarExpr, ok := fieldType.(*ast.StarExpr)
	if ok {
		// return fmt.Sprintf("*%v", p.getTypeAsString(astStarExpr.X))
		return fmt.Sprintf("%v", p.getTypeAsString(astStarExpr.X))
	}

	astSelectorExpr, ok := fieldType.(*ast.SelectorExpr)
	if ok {
		packageNameIdent, _ := astSelectorExpr.X.(*ast.Ident)
		return packageNameIdent.Name + "." + astSelectorExpr.Sel.Name
	}

	return fmt.Sprint(fieldType)
}

func parseStructTags(astField *ast.Field, structSchema *SchemaObject, fieldSchema *SchemaObject, name string) (newName string, skip bool) {
	if astField.Tag != nil {
		astFieldTag := reflect.StructTag(strings.Trim(astField.Tag.Value, "`"))
		tagText := ""

		if tag := astFieldTag.Get("goas"); tag != "" {
			tagText = tag
		}
		tagValues := strings.Split(tagText, ",")
		for _, v := range tagValues {
			if v == "-" {
				structSchema.DisabledFieldNames[name] = struct{}{}
				fieldSchema.Deprecated = true
				return "", true
			}
			parseTagValue := strings.Split(v, "=")
			if len(parseTagValue) > 0 {
				if parseTagValue[0] == "enum" {
					fieldSchema.Enum = strings.Split(parseTagValue[1], " ")
				}
			}
		}

		if tag := astFieldTag.Get("json"); tag != "" {
			tagText = tag
		}
		tagValues = strings.Split(tagText, ",")
		isRequired := false
		for _, v := range tagValues {
			if v == "-" {
				structSchema.DisabledFieldNames[name] = struct{}{}
				fieldSchema.Deprecated = true
				return "", true
			} else if v == "required" {
				isRequired = true
			} else if v != "" && v != "required" && v != "omitempty" {
				name = v
			}
		}

		if tag := astFieldTag.Get("example"); tag != "" {
			if fieldSchema.Type == nil {
				fieldSchema.Example = tag
			} else {
				switch *fieldSchema.Type {
				case "boolean":
					fieldSchema.Example, _ = strconv.ParseBool(tag)
				case "integer":
					fieldSchema.Example, _ = strconv.Atoi(tag)
				case "number":
					fieldSchema.Example, _ = strconv.ParseFloat(tag, 64)
				case "array":
					b, err := json.RawMessage(tag).MarshalJSON()
					if err != nil {
						fieldSchema.Example = "invalid example"
					} else {
						sliceOfInterface := []interface{}{}
						err := json.Unmarshal(b, &sliceOfInterface)
						if err != nil {
							fieldSchema.Example = "invalid example"
						} else {
							fieldSchema.Example = sliceOfInterface
						}
					}
				case "object":
					b, err := json.RawMessage(tag).MarshalJSON()
					if err != nil {
						fieldSchema.Example = "invalid example"
					} else {
						mapOfInterface := map[string]interface{}{}
						err := json.Unmarshal(b, &mapOfInterface)
						if err != nil {
							fieldSchema.Example = "invalid example"
						} else {
							fieldSchema.Example = mapOfInterface
						}
					}
				default:
					fieldSchema.Example = tag
				}
			}
		}
		if _, ok := astFieldTag.Lookup("required"); ok || isRequired {
			structSchema.Required = append(structSchema.Required, name)
		}

		if desc := astFieldTag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}
	}

	return name, false
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

// checkCache loops over possible aliased package names for a type to see if it's already in cache and returns that if found.
func (p *parser) checkCache(pkgName, typeName string) *SchemaObject {
	currentName := genSchemaObjectID(pkgName, typeName, p.PackageAliases)
	if knownObj, ok := p.KnownIDSchema[currentName]; ok {
		return knownObj
	} else if knownObj, ok := p.KnownIDSchema[typeName]; ok {
		return knownObj
	}

	for _, v := range p.PackageAliases {
		splitName := strings.Split(currentName, ".")
		if len(splitName) == 1 {
			newName := v + splitName[0]
			if foundObject, ok := p.KnownIDSchema[newName]; ok {
				return foundObject
			}
		} else if len(splitName) == 2 {
			newName := v + splitName[1]
			if foundObject, ok := p.KnownIDSchema[newName]; ok {
				return foundObject
			}
		}

		typeNameParts := strings.Split(typeName, ".")
		typeAlias := v + "." + typeNameParts[len(typeNameParts)-1]
		if foundObject, ok := p.KnownIDSchema[typeAlias]; ok {
			return foundObject
		}

	}
	return nil

}

func sortedPackageKeys(m map[string]*ast.Package) []string {
	keys := make([]string, len(m))
	i := 0
	for k, _ := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func sortedFileKeys(m map[string]*ast.File) []string {
	keys := make([]string, len(m))
	i := 0
	for k, _ := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

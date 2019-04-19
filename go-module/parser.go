// Package module is based on https://github.com/uudashr/go-module
package module

import (
	"fmt"
)

// Module represents the mod file.
type Module struct {
	Name     string       // Name of module
	Requires []Package    // Require declaration
	Excludes []Package    // Exclude declaration
	Replaces []PackageMap // Replace declaration
}

// PackageMap package mapping definition.
type PackageMap struct {
	From Package // Original package
	To   Package // Destination package
}

// Package represents the package info.
type Package struct {
	Path    string // Import path
	Version string // Version (semver)
}

// Parse module file from given b.
func Parse(b []byte) (*Module, error) {
	f := &Module{}
	l := lex(b)
	p := &parser{lexer: l, file: f}

	for state := parseModule; state != nil; {
		state = state(p)
	}

	if p.err != nil {
		return nil, p.err
	}

	return f, nil
}

// ParseInString module file from input string.
func ParseInString(s string) (*Module, error) {
	return Parse([]byte(s))
}

type parser struct {
	lexer *lexer
	file  *Module
	err   error
}

func (p *parser) nextToken() token {
	return p.lexer.nextToken()
}

func (p *parser) skipNewline() token {
	for {
		switch t := p.nextToken(); t.kind {
		case tokenNewline:
			// ignore
		case tokenEOF:
			fallthrough
		default:
			return t
		}
	}
}

func (p *parser) error(err error) parseFn {
	p.err = err
	return nil
}

func (p *parser) errorf(format string, args ...interface{}) parseFn {
	return p.error(fmt.Errorf(format, args...))
}

func (p *parser) goVersion(v string) {
	// fmt.Println("go version", v)
}

func (p *parser) requirePkg(pkg Package) {
	p.file.Requires = append(p.file.Requires, pkg)
}

func (p *parser) excludePkg(pkg Package) {
	p.file.Excludes = append(p.file.Excludes, pkg)
}

func (p *parser) replacePkg(m PackageMap) {
	p.file.Replaces = append(p.file.Replaces, m)
}

type parseFn func(p *parser) parseFn

func parseModule(p *parser) parseFn {
Loop:
	for {
		switch t := p.nextToken(); t.kind {
		case tokenNewline:
			// skip
		case tokenModule:
			break Loop
		default:
			return p.errorf("expect module declaration, got %s", t)
		}
	}

	return parseModuleName
}

func parseModuleName(p *parser) parseFn {
	t := p.nextToken()
	if t.kind != tokenNakedVal {
		return p.errorf("expect module name, got %s", t)
	}

	p.file.Name = t.val

	if t = p.nextToken(); t.kind != tokenNewline {
		return p.errorf("expect newline, got %s", t)
	}
	return parseVerb
}

func parseVerb(p *parser) parseFn {
	switch t := p.nextToken(); t.kind {
	case tokenRequire:
		return parsePkgList(p.requirePkg)
	case tokenExclude:
		return parsePkgList(p.excludePkg)
	case tokenReplace:
		return parsePkgMapList(p.replacePkg)
	case tokenGo:
		return parseGoVersion(p.goVersion)
	case tokenNewline:
		// ignore
		return parseVerb
	case tokenEOF:
		return nil
	default:
		return p.errorf("expect verb declaration, got %s", t)
	}
}

func parseGoVersion(add func(pkg string)) parseFn {
	return func(p *parser) parseFn {
		t := p.nextToken()
		add(t.val)
		return parseVerb
	}
}

func parsePkgList(add func(pkg Package)) parseFn {
	return func(p *parser) parseFn {
		t := p.nextToken()
		if t.kind == tokenLeftParen {
			if t = p.nextToken(); t.kind != tokenNewline {
				return p.errorf("expect newline, got %s", t)
			}

			return parsePkgListElem(add)
		}

		pkg, err := readPkg(t, p)
		if err != nil {
			return p.error(err)
		}

		if t = p.nextToken(); t.kind != tokenNewline {
			if t.kind != tokenIndirect {
				return p.errorf("expect newline, got %s", t)
			}
			return p.errorf("expect newline, got %s", t)
		}

		add(*pkg)
		return parseVerb
	}
}

func parsePkgListElem(add func(pkg Package)) parseFn {
	return func(p *parser) parseFn {
		t := p.skipNewline()
		if t.kind == tokenRightParen {
			if t = p.nextToken(); t.kind != tokenNewline {
				return p.errorf("expect newline, got %s", t)
			}

			return parseVerb
		}

		pkg, err := readPkg(t, p)
		if err != nil {
			return p.error(err)
		}

		if t = p.nextToken(); t.kind != tokenNewline {
			if t.kind != tokenIndirect {
				return p.errorf("expect newline, got %s", t)
			}
		}

		add(*pkg)
		return parsePkgListElem(add)
	}
}

func parsePkgMapList(add func(m PackageMap)) parseFn {
	return func(p *parser) parseFn {
		t := p.nextToken()
		if t.kind == tokenLeftParen {
			if t = p.nextToken(); t.kind != tokenNewline {
				return p.errorf("expect newline, got %s", t)
			}

			return parsePkgMapListElem(add)
		}

		pkgMap, err := readPkgMap(t, p)
		if err != nil {
			return p.error(err)
		}

		if t = p.nextToken(); t.kind != tokenNewline {
			return p.errorf("expect newline, got %s", t)
		}

		add(*pkgMap)
		return parseVerb
	}
}

func parsePkgMapListElem(add func(m PackageMap)) parseFn {
	return func(p *parser) parseFn {
		t := p.nextToken()
		if t.kind == tokenRightParen {
			if t = p.nextToken(); t.kind != tokenNewline {
				return p.errorf("expect newline, got %s", t)
			}

			return parseVerb
		}

		pkgMap, err := readPkgMap(t, p)
		if err != nil {
			return p.error(err)
		}

		if t = p.nextToken(); t.kind != tokenNewline {
			return p.errorf("expect newline, got %s", t)
		}

		add(*pkgMap)
		return parsePkgMapListElem(add)
	}
}

func readPkg(t token, p *parser) (*Package, error) {
	if t.kind != tokenNakedVal {
		return nil, fmt.Errorf("expect package declaration, got %s", t)
	}

	path := t.val

	if t = p.nextToken(); t.kind != tokenNakedVal {
		return nil, fmt.Errorf("expect package version, got %s", t)
	}

	return &Package{Path: path, Version: t.val}, nil
}

func readPkgMap(t token, p *parser) (*PackageMap, error) {
	old, err := readPkg(t, p)
	if err != nil {
		return nil, err
	}

	if t := p.nextToken(); t.kind != tokenMapFun {
		return nil, fmt.Errorf("expect '=>', got %s", t)
	}

	new, err := readPkg(p.nextToken(), p)
	if err != nil {
		return nil, err
	}

	return &PackageMap{From: *old, To: *new}, nil
}

// Package module is based on https://github.com/uudashr/go-module
package module

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const eof = rune(-1)

type tokenKind int

const (
	// special kind
	tokenError tokenKind = iota // error
	tokenEOF                    // end of file

	// operators
	tokenMapFun // "=>"

	// delimiters
	tokenLeftParen  // "("
	tokenRightParen // ")"
	tokenNewline    // "\n"

	// literals
	tokenNakedVal // naked value (string like, without double quote)

	// keywords
	tokenModule   // module
	tokenRequire  // require
	tokenExclude  // exclude
	tokenReplace  // replace
	tokenGo       // go
	tokenIndirect // indirect
)

var key = map[string]tokenKind{
	"module":   tokenModule,
	"require":  tokenRequire,
	"exclude":  tokenExclude,
	"replace":  tokenReplace,
	"go":       tokenGo,
	"indirect": tokenIndirect,
}

type token struct {
	kind tokenKind
	val  string
}

func (t token) String() string {
	switch t.kind {
	case tokenEOF:
		return "EOF"
	case tokenError:
		return t.val
	case tokenNewline:
		return "newline"
	}

	if len(t.val) > 10 {
		return fmt.Sprintf("%.10q...", t.val)
	}
	return fmt.Sprintf("%q", t.val)
}

type lexFn func(l *lexer) lexFn

type lexer struct {
	start  int        // start position of the token
	pos    int        // current read position of the input
	width  int        // width of the last runes read from the input
	input  []byte     // the input bytes being scanned
	tokens chan token // the scanned tokens
	state  lexFn      // the current state of lexer
}

func lex(b []byte) *lexer {
	l := &lexer{
		input:  b,
		tokens: make(chan token, 2),
		state:  lexFile,
	}
	return l
}

func lexInString(s string) *lexer {
	return lex([]byte(s))
}

func (l *lexer) nextToken() token {
	for {
		select {
		case t, ok := <-l.tokens:
			if !ok {
				return token{kind: tokenError, val: "no more token"}
			}

			if t.kind == tokenEOF {
				close(l.tokens)
			}
			return t
		default:
			l.state = l.state(l)
		}
	}
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRune(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) val() string {
	return string(l.input[l.start:l.pos])
}

func (l *lexer) emit(kind tokenKind) {
	i := token{kind: kind, val: l.val()}
	l.tokens <- i
	l.start = l.pos
}

func (l *lexer) emitErrorf(format string, args ...interface{}) lexFn {
	l.tokens <- token{kind: tokenError, val: fmt.Sprintf(format, args...)}
	return nil
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func lexFile(l *lexer) lexFn {
	for {
		switch r := l.next(); {
		case isWhiteSpace(r):
			l.ignore()
		case r == '\n':
			l.emit(tokenNewline)
			return lexFile
		case r == '"':
			return lexString
		case r == '(':
			l.emit(tokenLeftParen)
			return lexFile
		case r == ')':
			l.emit(tokenRightParen)
			return lexFile
		case r == '=':
			if l.next() != '>' {
				return l.emitErrorf("expect => got %q", string(r))
			}

			l.emit(tokenMapFun)
			return lexFile
		case isAlpha(r):
			return lexKeywordOrNakedVal
		case r >= '0' && r <= '9':
			return lexKeywordOrNakedVal
		case r == '_':
			return lexKeywordOrNakedVal
		case r == '/':
			l.ignore()
		case r == eof:
			l.ignore()
			l.emit(tokenEOF)
			return nil
		default:

			return l.emitErrorf("expecting valid keyword while lexFile, got %q", string(r))
		}
	}
}

func lexKeywordOrNakedVal(l *lexer) lexFn {
	for {
		switch r := l.next(); {
		case unicode.IsLetter(r), unicode.IsDigit(r), strings.ContainsRune("+-./_", r):
			// absorb
		default:
			l.backup()
			word := l.val()
			if kind, ok := key[word]; ok {
				l.emit(kind)
				return lexFile
			}

			l.emit(tokenNakedVal)
			return lexFile
		}
	}
}

func lexString(l *lexer) lexFn {
	for {
		switch r := l.next(); {
		case r == '\n', r == eof:
			return l.emitErrorf("unterminated string, got %s", string(r))
		case r == '\\':
			r = l.next()
			if !(r == 't' || r == '\\') {
				return l.emitErrorf(`invalid escape char \%s`, string(r))
			}
			fallthrough
		default:
			// absorp
		}
	}
}

func isWhiteSpace(r rune) bool {
	return strings.ContainsRune(" \t", r)
}

func isAlpha(r rune) bool {
	return unicode.IsLetter(r)
}

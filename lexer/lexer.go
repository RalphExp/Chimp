package lexer

import (
	"bufio"
	"chimp/token"
	"fmt"
	"io"
	"strings"
)

type Lexer struct {
	scanner  *bufio.Scanner
	input    string
	position int    // current position in input (points to current char)
	ch       byte   // current char under examination
	file     string // current file name or (stdin) if not read from file
	line     int    // line number
	column   int    // column number
	repl     bool
}

/* XXX: the original version of the lexer read too many characters ahead,
 * since we have to implement a better REPL, we should let the lexer read
 * as less as possible.*/
func New(reader io.Reader) *Lexer {
	l := &Lexer{
		scanner:  bufio.NewScanner(reader),
		position: 0,
		repl:     true,
	}
	return l
}

func NewString(input string) *Lexer {
	reader := strings.NewReader(input)
	l := &Lexer{
		scanner:  bufio.NewScanner(reader),
		position: 0,
		repl:     false,
	}
	return l
}

func NewFile(input string) *Lexer {
	return nil
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.readChar()
	l.skipWhitespace()

	switch l.ch {
	case '=':
		if l.getChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: "=="}
		} else {
			tok = newToken(token.ASSIGN, '=')
		}
	case '+':
		if l.getChar() == '=' {
			l.readChar()
			literal := "+="
			tok = token.Token{Type: token.ADD_ASSIGN, Literal: literal}
		} else {
			tok = newToken(token.PLUS, '+')
		}
	case '-':
		if l.getChar() == '=' {
			l.readChar()
			literal := "-="
			tok = token.Token{Type: token.SUB_ASSIGN, Literal: literal}
		} else {
			tok = newToken(token.MINUS, '-')
		}
	case '!':
		if l.getChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: "!="}
		} else {
			tok = newToken(token.BANG, '!')
		}
	case '/':
		switch l.getChar() {
		case '=':
			l.readChar() // advance pointer
			tok = token.Token{Type: token.DIV_ASSIGN, Literal: "/="}
		case '/':
			// line comment
			// discard the rest characters in the buffer
			l.position = len(l.input)
			tok = token.Token{Type: token.COMMENT, Literal: "//"}
		case '*':
			// block comment handling
			// eats '*'
			l.readChar()
			for {
				ch := l.readChar()
				if ch == 0 {
					break
				}
				if ch != '*' {
					continue
				}
				ch = l.readChar()
				if ch == 0 || ch == '/' {
					break
				}
			}
			tok = token.Token{Type: token.COMMENT, Literal: "/**/"}
		default:
			tok = newToken(token.DIV, '/')
		}
	case '*':
		if l.getChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.MUL_ASSIGN, Literal: "*="}
		} else {
			tok = newToken(token.MUL, '*')
		}
	case '%':
		if l.getChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.MOD_ASSIGN, Literal: "%="}
		} else {
			tok = newToken(token.MOD, '%')
		}
	case '<':
		tok = newToken(token.LT, l.ch)
	case '>':
		tok = newToken(token.GT, l.ch)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
	case ':':
		tok = newToken(token.COLON, l.ch)
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '{':
		tok = newToken(token.LBRACE, l.ch)
	case '}':
		tok = newToken(token.RBRACE, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
		if l.ch != '"' {
			tok.Type = token.ILLEGAL
			tok.Literal = "no right \" found"
		} else {
			l.readChar() // eats '"'
		}
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}

	return tok
}

func (l *Lexer) GetInput() string {
	return fmt.Sprintf("input: %s, pos: %d\n", l.input, l.position)
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) getChar() byte {
	l.readNext(false)
	return l.ch
}

func (l *Lexer) readChar() byte {
	l.readNext(true)
	return l.ch
}

func (l *Lexer) readNext(inc bool) {
	if l.position >= len(l.input) {
		for l.scanner.Scan() {
			if len(l.scanner.Text()) > 0 {
				l.input = l.scanner.Text()
				l.position = 0
				break
			}
		}
	}

	if l.position >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.position]
	}
	if inc {
		l.position++
	}
}

func (l *Lexer) readIdentifier() string {
	id := []byte{l.ch}
	for isLetter(l.getChar()) || isDigit(l.getChar()) {
		id = append(id, l.readChar())
	}
	return string(id)
}

func (l *Lexer) readNumber() string {
	num := []byte{l.ch}

	for isDigit(l.getChar()) {
		num = append(num, l.readChar())
	}

	return string(num)
}

func (l *Lexer) readString() string {
	str := []byte{}
	for {
		if l.getChar() == '"' || l.getChar() == 0 {
			break
		}
		str = append(str, l.readChar())
	}
	return string(str)
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}

package lexer

import (
	"bufio"
	"chimp/token"
	"io"
	"strconv"
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
 * as less as possible for further parsing by the parser. */
func New(reader io.Reader) *Lexer {
	l := &Lexer{
		scanner:  bufio.NewScanner(reader),
		position: 0,
		repl:     true,
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
			literal := "=="
			tok = token.Token{Type: token.EQ, Literal: literal}
		} else {
			tok = newToken(token.ASSIGN, '=')
		}
	case '+':
		tok = newToken(token.PLUS, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '!':
		if l.getChar() == '=' {
			l.readChar()
			literal := "!="
			tok = token.Token{Type: token.NOT_EQ, Literal: literal}
		} else {
			tok = newToken(token.BANG, '!')
		}
	case '/':
		tok = newToken(token.SLASH, l.ch)
	case '*':
		tok = newToken(token.ASTERISK, l.ch)
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
			// fmt.Printf("token: %s\n", tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			// fmt.Printf("token: %s\n", tok.Literal)
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}

	// fmt.Printf("token: %s\n", tok.Literal)
	return tok
}

func (l *Lexer) GetInput() string {
	return l.input + "," + strconv.Itoa(l.position)
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
				l.input += l.scanner.Text()
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

func (l *Lexer) peekChar() byte {
	if l.position+1 >= len(l.input) {
		for l.scanner.Scan() {
			if len(l.scanner.Text()) > 0 {
				l.input += l.scanner.Text()
				break
			}
		}
	}

	if l.position+1 >= len(l.input) {
		return 0
	} else {
		return l.input[l.position+1]
	}
	// don't advance position
}

func (l *Lexer) readIdentifier() string {
	id := []byte{l.ch}
	for isLetter(l.getChar()) {
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

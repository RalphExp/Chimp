package token

type TokenType int

const (
	NIL = iota
	ILLEGAL
	EOF
	// Identifiers + literals
	IDENT
	INT
	// "foobar"
	STRING
	// Operators
	ASSIGN
	PLUS
	MINUS
	BANG     // !
	ASTERISK // "*"
	SLASH    // "/"
	LT       // "<"
	GT       // ">"
	EQ       // "=="
	NOT_EQ   // "!="
	// Delimiters
	COMMA     // ","
	SEMICOLON // ";"
	COLON     // ":"
	LPAREN    // "("
	RPAREN    // ")"
	LBRACE    // "{"
	RBRACE    // "}"
	LBRACKET  // "["
	RBRACKET  // "]"
	// Keywords
	FUNCTION
	LET
	TRUE
	FALSE
	IF
	ELSE
	RETURN
	// Added in Chimp Parser
	DO
	FOR
	WHILE
	BREAK
	CONTINUE
)

type Token struct {
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"func":     FUNCTION,
	"let":      LET,
	"true":     TRUE,
	"null":     NIL,
	"false":    FALSE,
	"if":       IF,
	"else":     ELSE,
	"return":   RETURN,
	"for":      FOR,
	"do":       DO,
	"while":    WHILE,
	"break":    BREAK,
	"continue": CONTINUE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

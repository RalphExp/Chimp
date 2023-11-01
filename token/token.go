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
	ADD_ASSIGN
	SUB_ASSIGN
	MUL_ASSIGN
	DIV_ASSIGN
	MOD_ASSIGN
	INC
	DEC
	PLUS
	MINUS
	BANG   // !
	MUL    // "*"
	DIV    // "/"
	MOD    // "%"
	LT     // "<"
	GT     // ">"
	EQ     // "=="
	NOT_EQ // "!="
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
	COMMENT
	NULL
)

type Token struct {
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"func":     FUNCTION,
	"let":      LET,
	"true":     TRUE,
	"null":     NULL,
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

var token2name = map[int]string{
	NULL:       "null",
	ILLEGAL:    "illegal",
	EOF:        "eof",
	IDENT:      "id",
	INT:        "int",
	STRING:     "string",
	ASSIGN:     "=",
	ADD_ASSIGN: "+=",
	SUB_ASSIGN: "-=",
	MUL_ASSIGN: "*=",
	DIV_ASSIGN: "/=",
	MOD_ASSIGN: "%=",
	INC:        "++",
	DEC:        "--",
	PLUS:       "+",
	MINUS:      "-",
	BANG:       "!",
	MUL:        "*",
	DIV:        "/",
	MOD:        "%",
	LT:         "<",
	GT:         ">",
	EQ:         "==",
	NOT_EQ:     "!=",
	COMMA:      ",",
	SEMICOLON:  ";",
	COLON:      ":",
	LPAREN:     "(",
	RPAREN:     ")",
	LBRACE:     "{",
	RBRACE:     "}",
	LBRACKET:   "[",
	RBRACKET:   "]",
	FUNCTION:   "func",
	LET:        "let",
	TRUE:       "true",
	FALSE:      "false",
	IF:         "if",
	ELSE:       "else",
	RETURN:     "return",
	DO:         "do",
	FOR:        "for",
	WHILE:      "while",
	BREAK:      "break",
	CONTINUE:   "continue",
	COMMENT:    "comment",
}

func (t TokenType) Name() string {
	return token2name[int(t)]
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

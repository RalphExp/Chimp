package parser

import (
	"chimp/ast"
	"chimp/lexer"
	"chimp/token"
	"fmt"
	"strconv"
)

const (
	_ int = iota * 10
	LOWEST
	ASSIGN      // =|+=|-=|*=|/=|%=
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

var precedences = map[token.TokenType]int{
	token.ASSIGN:     ASSIGN,
	token.ADD_ASSIGN: ASSIGN,
	token.SUB_ASSIGN: ASSIGN,
	token.MUL_ASSIGN: ASSIGN,
	token.DIV_ASSIGN: ASSIGN,
	token.MOD_ASSIGN: ASSIGN,
	token.EQ:         EQUALS,      // == 2
	token.NOT_EQ:     EQUALS,      // != 2
	token.LT:         LESSGREATER, // <  3
	token.GT:         LESSGREATER, // >  3
	token.PLUS:       SUM,         // +  4
	token.MINUS:      SUM,         // -  4
	token.MUL:        PRODUCT,     // /  5
	token.DIV:        PRODUCT,     // *  5
	token.MOD:        PRODUCT,     // %  5
	token.LPAREN:     CALL,        // () 7
	token.LBRACKET:   INDEX,       // [] 8
}

var assignmentOp = map[string]bool{
	"=":  true,
	"+=": true,
	"-=": true,
	"*=": true,
	"/=": true,
	"%=": true,
}

func IsAssignmentOperator(op string) bool {
	_, ok := assignmentOp[op]
	return ok
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.NULL, p.parseNull)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.ADD_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.SUB_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.MUL_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.DIV_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.MOD_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.MUL, p.parseInfixExpression)
	p.registerInfix(token.DIV, p.parseInfixExpression)
	p.registerInfix(token.MOD, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)

	return p
}

func (p *Parser) NextToken() {
	p.nextToken()
}

func (p *Parser) GetToken() token.Token {
	if p.curToken.Type == token.NIL {
		p.curToken = p.l.NextToken()
		p.peekToken.Type = token.NIL
	}
	return p.curToken
}

func (p *Parser) PeekToken() token.Token {
	p.GetToken()
	if p.peekToken.Type == token.NIL {
		p.peekToken = p.l.NextToken()
	}
	return p.peekToken
}

func (p *Parser) nextToken() {
	if p.peekToken.Type != token.NIL {
		p.curToken = p.peekToken
	} else {
		p.curToken = p.l.NextToken()
	}
	p.peekToken.Type = token.NIL
	p.peekToken.Literal = ""
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	p.GetToken()
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	p.PeekToken()
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be '%s', got '%s' instead",
		t.Name(), p.peekToken.Type.Name())
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for '%s' found", t.Name())
	p.errors = append(p.errors, msg)
}

func (p *Parser) notMatchError(t token.TokenType) {
	msg := fmt.Sprintf("token not match, expect '%s', but got '%s'",
		t.Name(), p.curToken.Type.Name())
	p.errors = append(p.errors, msg)
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) ParseStatement() ast.Statement {
	return p.parseStatement()
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.GetToken().Type {
	case token.LET:
		return p.parseLetStatement()
	case token.IF:
		return p.parseIfStatment()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.FOR:
		return p.parseForStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.DO:
		return p.parseDoWhileStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.SEMICOLON:
		fallthrough
	case token.EOF:
		return nil
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.GetToken()}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.GetToken(), Value: p.GetToken().Literal}

	if !p.peekTokenIs(token.ASSIGN) {

		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		// support let a;
		// which will create an elemnet in the environment with a NULL value
		return stmt
	}

	p.nextToken() // eats IDENT
	p.nextToken() // eats '='

	stmt.Value = p.parseExpression(LOWEST)

	// if fl, ok := stmt.Value.(*ast.FunctionLiteral); ok {
	// 	fl.Name = stmt.Name.Value
	// }

	if fl, ok := stmt.Value.(*ast.FunctionLiteral); ok {
		if fl.Name == "" {
			fl.Name = stmt.Name.Value
		}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.GetToken()}

	// support return statement without value, in this case
	// nil will be implicitly as a return value
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
		stmt.ReturnValue = &ast.Null{}
		return stmt
	}

	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.GetToken()}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.GetToken().Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.GetToken().Type)
		return nil
	}
	leftExp := prefix()

	// assign is right association, need to decrease precedence
	if precedence == ASSIGN {
		precedence--
	}

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.PeekToken().Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.PeekToken().Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.GetToken().Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.GetToken(), Value: p.GetToken().Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.GetToken()}

	value, err := strconv.ParseInt(p.GetToken().Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.GetToken().Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value

	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.GetToken(), Value: p.GetToken().Literal}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.GetToken(),
		Operator: p.GetToken().Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.GetToken(),
		Operator: p.GetToken().Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseNull() ast.Expression {
	return &ast.Null{Token: p.GetToken()}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.GetToken(), Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Token: p.GetToken()}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{Token: p.GetToken()}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStatement {
	return nil
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	statement := &ast.WhileStatement{Token: p.GetToken()}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	// expectPeek eats 'while' and nextToken eats '('
	p.nextToken()

	statement.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		statement.Body = p.parseBlockStatement()
	} else {
		p.nextToken()
		statement.Body = p.parseStatement()
	}

	return statement
}

func (p *Parser) parseDoWhileStatement() ast.Statement {
	statement := &ast.DoWhileStatement{Token: p.GetToken()}

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		statement.Body = p.parseBlockStatement()
	} else {
		p.nextToken()
		statement.Body = p.parseStatement()
	}

	if !p.expectPeek(token.WHILE) {
		return nil
	}
	// now current token is WHILE

	if !p.expectPeek(token.LPAREN) {
		// expectPeek eats WHILE
		return nil
	}

	// eats '('
	p.nextToken()
	statement.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return statement
}

func (p *Parser) parseIfStatment() ast.Statement {
	statement := &ast.IfStatement{Token: p.GetToken()}

	if !p.expectPeek(token.LPAREN) { // eats if
		return nil
	}

	p.nextToken() // eats '('
	statement.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		statement.Consequence = p.parseBlockStatement()
	} else {
		p.nextToken()
		statement.Consequence = p.parseStatement()
	}

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if p.peekTokenIs(token.LBRACE) {
			p.nextToken()
			statement.Alternative = p.parseBlockStatement()
		} else {
			p.nextToken()
			statement.Alternative = p.parseStatement()
		}
	}

	return statement
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.GetToken()}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	if !p.curTokenIs(token.RBRACE) {
		p.notMatchError(token.RBRACE)
	}
	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.GetToken()}

	if p.peekTokenIs(token.IDENT) {
		lit.Name = p.PeekToken().Literal
		p.nextToken()
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.GetToken(), Value: p.GetToken().Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.GetToken(), Value: p.GetToken().Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.GetToken(), Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.GetToken()}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.GetToken(), Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.GetToken()}
	hash.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		hash.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

package parser

import (
	"chimp/ast"
	"chimp/lexer"
	"chimp/token"
	"fmt"
	"strconv"
)

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

var precedences = map[token.TokenType]int{
	token.EQ:       EQUALS,      // == 2
	token.NOT_EQ:   EQUALS,      // != 2
	token.LT:       LESSGREATER, // <  3
	token.GT:       LESSGREATER, // >  3
	token.PLUS:     SUM,         // +  4
	token.MINUS:    SUM,         // -  4
	token.SLASH:    PRODUCT,     // /  5
	token.ASTERISK: PRODUCT,     // *  5
	token.LPAREN:   CALL,        // () 7
	token.LBRACKET: INDEX,       // [] 8
}

var assignmentOp = map[token.TokenType]bool{
	token.ASSIGN:     true,
	token.ADD_ASSIGN: true,
	token.SUB_ASSIGN: true,
	token.MUL_ASSIGN: true,
	token.DIV_ASSIGN: true,
	token.MOD_ASSIGN: true,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken_  token.Token
	peekToken_ token.Token

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
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
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
	if p.curToken_.Type == token.NIL {
		p.curToken_ = p.l.NextToken()
		p.peekToken_.Type = token.NIL
	}
	return p.curToken_
}

func (p *Parser) PeekToken() token.Token {
	p.GetToken()
	if p.peekToken_.Type == token.NIL {
		p.peekToken_ = p.l.NextToken()
	}
	return p.peekToken_
}

func (p *Parser) nextToken() {
	if p.peekToken_.Type != token.NIL {
		p.curToken_ = p.peekToken_
	} else {
		p.curToken_ = p.l.NextToken()
	}
	p.peekToken_.Type = token.NIL
	p.peekToken_.Literal = ""
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	p.GetToken()
	return p.curToken_.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	p.PeekToken()
	return p.peekToken_.Type == t
}

func (p *Parser) peekTokenIsAssignment() bool {
	p.PeekToken()
	_, ok := assignmentOp[p.PeekToken().Type]
	return ok
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
	msg := fmt.Sprintf("expected next token to be %d, got %d instead",
		t, p.peekToken_.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %d found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) notMatchError(t token.TokenType) {
	msg := fmt.Sprintf("token not match, expect '%d', but got %d", t, p.curToken_.Type)
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
	// chimp's new feature
	case token.WHILE:
		return p.parseWhileStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.SEMICOLON:
		fallthrough
	case token.EOF:
		return nil
	case token.IDENT:
		if p.peekTokenIsAssignment() {
			return p.parseAssignmentStatement()
		}
		fallthrough
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseAssignmentStatement() *ast.AssignmentStatement {
	stmt := &ast.AssignmentStatement{
		Token: p.GetToken(),
		Name:  &ast.Identifier{Token: p.GetToken(), Value: p.GetToken().Literal}}

	p.nextToken()

	stmt.Operator = p.GetToken().Literal

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

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

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	statement := &ast.WhileStatement{Token: p.GetToken()}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken() // eats '('
	statement.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		statement.Statement = p.parseBlockStatement()
	} else {
		p.nextToken()
		statement.Statement = p.parseStatement()
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
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

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
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

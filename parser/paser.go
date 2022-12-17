package parser

import (
	"fmt"
	"lexer-parser/ast"
	"lexer-parser/lexer"
	"lexer-parser/token"
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

// Precedence table and a few helper methods
//
//	it associates token types with their precedence.
var precedences = map[token.TokenType]int{
	// "==" or "="
	token.EQ: EQUALS,
	// "!="
	token.NOT_EQ: EQUALS,
	// "<"
	token.LT: LESSGREATER,
	// ">"
	token.GT: LESSGREATER,
	// "+"
	token.PLUS: SUM,
	// "-"
	token.MINUS: SUM,
	// "/"
	token.SLASH: PRODUCT,
	// "*"
	token.ASTERISK: PRODUCT,
	// "("
	token.LPAREN: CALL,

	token.LBRACKET: INDEX,
}

// peek the peekToken Type for which precedence else return LOWEST
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// check current Token Type for which precedence else return LOWEST
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// parse the tokens just in time
// convert tokens -> ast nodes
type Parser struct {
	l      *lexer.Lexer // a pointer to an instance of the lexer
	errors []string

	curToken  token.Token // like in the lexer - position
	peekToken token.Token // like in the lexer - readPosition

	// prefix and infix Parser functions mapper
	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// if we encounter a token of type token.
	// IDENT the parsing function to call is parseIdentifier,
	// a method we defined on *Parser
	// a set of parsers for <prefix operator> <express>
	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	// <identifier literal>
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	// <integer literal>
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	// <!> <expression>
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	// <-> <expression>
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)

	// <true>
	p.registerPrefix(token.TRUE, p.parseBoolean)
	// <false>
	p.registerPrefix(token.FALSE, p.parseBoolean)

	// <(> <expression> )
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)

	// <if> ( <condition> ) { <consequence> } [else { alternative }]
	// not <else if>
	p.registerPrefix(token.IF, p.parseIfExpression)

	// <fn>
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)

	// <"> <literal> "
	p.registerPrefix(token.STRING, p.parseStringLiteral)

	// <[> <literal> ]
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)

	p.registerPrefix(token.LBRACE, p.parseHashLiteral)

	// Every infix operator gets associated with the same parsing function called parseInfixExpression
	// a set of parsers for <expression> <infix operator> <expression>
	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	// <expression> + <expression>
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	// <expression> - <expression>
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	// <expression> / <expression>
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	// <expression> * <expression>
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	// <expression> == <expression>
	p.registerInfix(token.EQ, p.parseInfixExpression)
	// <expression> != <expression>
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	// <expression> < <expression>
	p.registerInfix(token.LT, p.parseInfixExpression)
	// <expression> > <expression>
	p.registerInfix(token.GT, p.parseInfixExpression)
	// <expression> ( <expression> )
	p.registerInfix(token.LPAREN, p.parseCallExpression)

	p.registerInfix(token.LBRACKET, p.parseIndexExpression)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// return the Parser errors[]
func (p *Parser) Errors() []string {
	return p.errors
}

// peek the expected token else append error to self errors[]
func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// judge the next token, if judgement is true then will forward a token
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// a small helper that advances both curToken and peekToken
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// Construct the root node of the AST, an *ast.Program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}
	// It then iterates over every token in the input until it encounters an token.EOF token.
	for p.curToken.Type != token.EOF {
		// parse a statement
		stmt := p.parseStatement()

		// if stmt != nil {
		program.Statements = append(program.Statements, stmt)
		// }

		p.nextToken()
	}
	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		// let <identifier literal> = <expression>;
		return p.parseLetStatement()
	case token.RETURN:
		// return <expression>;
		return p.parseReturnStatement()
	default:
		// <expression>;
		return p.parseExpressionStatement()
	}
}

// when it encounters a token.LET token
// parse statement `let (Name)<Identifier> = (Value)<expression>?;`
func (p *Parser) parseLetStatement() *ast.LetStatement {
	// initialize letStatement
	stmt := &ast.LetStatement{Token: p.curToken}

	// check next token whether is an Identifier token
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	// add it to the Name part
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// check next token whether is an ASSIGN("=") token
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	// skip the Assign token
	p.nextToken()

	// parse the Expression to self Value
	stmt.Value = p.parseExpression(LOWEST)

	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parse statement `return (value)<expression>?;`
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	// initialize the return statement
	stmt := &ast.ReturnStatement{Token: p.curToken}

	// skip "return"
	p.nextToken()

	// parse the next expression to self Value
	stmt.ReturnValue = p.parseExpression(LOWEST)

	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parse statement `<expression>;'
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	// initialize expression statement
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	// parse the expression to self Expression
	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parse statement which contains a series of statements
// <statement>*
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	// initialize block statements
	block := &ast.BlockStatement{Token: p.curToken}
	// initialize Statements array in block
	block.Statements = []ast.Statement{}

	// skip "{"
	p.nextToken()

	// check
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		block.Statements = append(block.Statements, stmt)
		p.nextToken()
	}
	return block
}

// parsing Integer literal
func (p *Parser) parseIntegerLiteral() ast.Expression {
	// initialize integer literals
	lit := &ast.IntegerLiteral{Token: p.curToken}

	// convert string to integer
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value

	return lit
}

// parse literal `fn ( (Parameters)<parameters,>* ) { (Body)<BlockStatement> }`
func (p *Parser) parseFunctionLiteral() ast.Expression {
	// initialize function literal
	lit := &ast.FunctionLiteral{Token: p.curToken}

	// check peek token whether is "("
	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	// parse parameters to Self Parameters
	lit.Parameters = p.parseFunctionParameters()

	// check peek token whether is "{"
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// parse block statement to Self Body
	lit.Body = p.parseBlockStatement()

	return lit
}

// parse helper for parseFunctionLiteral (`<parameter:Identifier(s)>`)
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	// initialize identifiers[]
	identifiers := []*ast.Identifier{}

	// check peekToken whether is ")" and skip it, meaning not parameter,
	// return an empty identifiers[]
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	// skip token ")"
	p.nextToken()

	// meaning having a least one identifier token
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	// append the first to the identifiers[]
	identifiers = append(identifiers, ident)

	// check the token "," , if exists meaning next token must be a identifier.
	for p.peekTokenIs(token.COMMA) {
		// skip ","
		p.nextToken()
		// cursor stop on the next identifier token
		p.nextToken()
		// initialize it ant then append to the identifiers[]
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	// check next token whether is ")"
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

// handle parsing expression by the Parser,
//
//	which from Self->prefixParseFns array and Self->infixPareFns array
//	 precedence means the  current token precedence
func (p *Parser) parseExpression(precedence int) ast.Expression {
	// take out the prefix parse fn from prefixParseFns mapper
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	// And it does all this again and
	// again until it encounters a token that has a lower precedence.
	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		// take out the infix parse fn from prefixParseFns mapper
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		// infix parse function , put lef-exp in as the Infix expression of Left
		leftExp = infix(leftExp)
	}
	return leftExp
}

// parse Prefix
// <prefix operator><expression>;
func (p *Parser) parsePrefixExpression() ast.Expression {
	// initialize prefix expression
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	// skip current token operator
	p.nextToken()

	// parse expression to Self Right, which the expression has the precedence of PREFIX
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// parse Infix expression
// (Left)<expression> (Operator)<infix operator> (Right)<expression>
// left is the pre Expression
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	// initialize infix expression
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	// get current token precedence from the table precedences[tokenType]int
	precedence := p.curPrecedence()
	// skip self, operator
	p.nextToken()
	// parse expression with self precedence to Right
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parse e.g. input : (5 + 5) * 2;
func (p *Parser) parseGroupedExpression() ast.Expression {
	// skip "("
	p.nextToken()

	// parse expression with LOWEST precedence
	exp := p.parseExpression(LOWEST)

	// check next token whether is ")"
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// parse if - else
// if ( (Condition)<blockStatement> )
//
// { (Consequence)<blockStatement> }
//
// else
//
// { (Alternative)<blockStatement> }
func (p *Parser) parseIfExpression() ast.Expression {
	// initialize if expression
	expression := &ast.IfExpression{Token: p.curToken}

	// check next token whether is "("
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	// skip "("
	p.nextToken()
	// parse expression to Condition with precedence LOWEST
	expression.Condition = p.parseExpression(LOWEST)

	// check next token whether is ")"
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	// check next token whether is "{"
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// parse BlockStatement to Consequence
	expression.Consequence = p.parseBlockStatement()

	// check next token whether is "else"
	if p.peekTokenIs(token.ELSE) {
		// skip "else"
		p.nextToken()

		// check next token whether is "{"
		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		// parse BlockStatement to Alternative
		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

// parse the call function
// "(function)<expression>( (Arguments[])<expression>* )"
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	// parse the arguments
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

// parse to a ast.Expression, form by current cursor token
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parse boolean value
func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}

// parse the a series of argument
// (Arguments[])<expression>*
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	// initialize the args expression[]
	list := []ast.Expression{}

	// check the next token whether is token end
	if p.peekTokenIs(end) {
		// skip the end token
		p.nextToken()
		return list
	}

	// skip the current token
	p.nextToken()
	// means there will be at least one argument
	// parse expression with precedence LOWEST, then append it to the args[]
	list = append(list, p.parseExpression(LOWEST))

	// check whether next token is ","
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()

		list = append(list, p.parseExpression(LOWEST))
	}

	// check the next token whether is end token
	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseHashLiteral() ast.Expression {
	// initialize hash literal
	hash := &ast.HashLiteral{Token: p.curToken}

	// initialize hash.Pairs which contain the [ast.Expression]:[ast.Expression]
	hash.Pairs = make(map[ast.Expression]ast.Expression)

	// until meet next token is "{"
	for !p.peekTokenIs(token.RBRACE) {
		// skip "{"
		p.nextToken()
		// parse the key expression
		key := p.parseExpression(LOWEST)
		// if the next token isn't ":"
		if !p.expectPeek(token.COLON) {
			return nil
		}
		// skip ":"
		p.nextToken()

		// parse the value expression
		value := p.parseExpression(LOWEST)
		// add the key-value to hash.Pairs
		hash.Pairs[key] = value
		// if next token isn't "}" and  next token also isn't ","
		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}
	// the parser end at the token "}"
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return hash
}

// Error handle for no prefix error
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

// check the current cursor token whether matches the want token type
func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

// check the peek cursor token whether matches the want token type
func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

type (
	// A prefix operator does not have left side
	// called when we encounter associated type token type in prefix position
	prefixParseFn func() ast.Expression

	// Function infixParseFn  param is the left side of the infix operator that's being parsed.
	// called when we encounter associated type token in infix position
	infixParseFn func(ast.Expression) ast.Expression
)

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

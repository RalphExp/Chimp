package compiler

import (
	"chimp/ast"
	"chimp/code"
	"chimp/object"
	"chimp/parser"
	"fmt"
	"sort"
)

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

type JmpContext struct {
	ips []int // instruction pointer
}

type Compiler struct {
	constants       []object.Object
	symbolTable     *SymbolTable
	scopes          []*CompilationScope
	breakContext    []JmpContext
	continueContext []JmpContext
	scopeIndex      int
}

func New() *Compiler {
	mainScope := &CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	// XXX: pass the compiler to symbol table??
	symbolTable := NewSymbolTable()

	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	return &Compiler{
		constants:       []object.Object{},
		symbolTable:     symbolTable,
		scopes:          []*CompilationScope{mainScope},
		scopeIndex:      0,
		breakContext:    make([]JmpContext, 0),
		continueContext: make([]JmpContext, 0),
	}
}

func (c *Compiler) pushBreakContext() {
	c.breakContext = append(c.breakContext, JmpContext{})
}

func (c *Compiler) popBreakContext() {
	l := len(c.breakContext)
	c.breakContext = c.breakContext[0 : l-1]
}

func (c *Compiler) pushContinueContext() {
	c.continueContext = append(c.continueContext, JmpContext{})
}

func (c *Compiler) popContinueContext() {
	l := len(c.continueContext)
	c.continueContext = c.continueContext[0 : l-1]
}

func NewWithState(s *SymbolTable, constants []object.Object) *Compiler {
	compiler := New()
	compiler.symbolTable = s
	compiler.constants = constants
	return compiler
}

func (c *Compiler) CompileAssignment(node *ast.InfixExpression) error {
	switch lhs := node.Left.(type) {
	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(lhs.Value)
		if !ok {
			return fmt.Errorf("undefined variable %s", lhs.Value)
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+=":
			c.emit(code.OpAdd)
		case "-=":
			c.emit(code.OpSub)
		case "*=":
			c.emit(code.OpMul)
		case "/=":
			c.emit(code.OpDiv)
		case "%=":
			c.emit(code.OpMod)
		default:
			/* do nothing */
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
			c.emit(code.OpGetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
			c.emit(code.OpGetLocal, symbol.Index)
		}

	default:
		return fmt.Errorf("invalid left hand side value in assignment")
	}
	return nil
}

func (c *Compiler) CompileBlockStatement(
	node *ast.BlockStatement,
	newFrame bool,
) error {
	if !newFrame {
		// if the block is not introduced by a function, i.e. by if, while, ... etc
		c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)

		if c.symbolTable.Outer.Outer == nil {
			// e.g. when the program is
			// let a = 1; if (1) { let a = 1; }
			// we want the COMMANDS to be SETLOCAL 0 rather than SETCLOCAL 1
			c.symbolTable.numDefinitions = 0
		} else {
			c.symbolTable.numDefinitions = c.symbolTable.Outer.numDefinitions
		}
		c.symbolTable.block = true

		defer func() {
			c.symbolTable = c.symbolTable.Outer
		}()
	}

	for _, s := range node.Statements {
		err := c.Compile(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) CompileLogicalOperator(node *ast.InfixExpression) error {
	var jumpToEnd int
	err := c.Compile(node.Left)
	if err != nil {
		return err
	}

	if node.Operator == "&&" {
		jumpToEnd = c.emit(code.OpJumpIfFalseNonPop, -1)
	} else if node.Operator == "||" {
		jumpToEnd = c.emit(code.OpJumpIfTrueNonPop, -1)
	} else {
		panic(fmt.Sprintf("unknow operator: %s\n", node.Operator))
	}

	c.emit(code.OpPop)
	err = c.Compile(node.Right)
	if err != nil {
		return err
	}

	end := len(c.currentInstructions())
	c.changeOperand(jumpToEnd, end)
	return nil
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop)

	case *ast.InfixExpression:
		if parser.IsAssignmentOperator(node.Operator) {
			return c.CompileAssignment(node)
		}

		if parser.IsLogicalOperator(node.Operator) {
			return c.CompileLogicalOperator(node)
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "%":
			c.emit(code.OpMod)
		case "<":
			c.emit(code.OpLess)
		case "<=":
			c.emit(code.OpLessEqual)
		case ">":
			c.emit(code.OpGreater)
		case ">=":
			c.emit(code.OpGreaterEqual)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.BreakStatement:
		if len(c.breakContext) == 0 {
			return fmt.Errorf("no break context found")
		}
		l := len(c.breakContext) - 1

		pos := c.emit(code.OpJump, -1)
		// later the pos will change duration backfill
		c.breakContext[l].ips = append(c.breakContext[l].ips, pos)

	case *ast.ContinueStatement:
		if len(c.continueContext) == 0 {
			return fmt.Errorf("no continue context found")
		}

		l := len(c.continueContext) - 1
		pos := c.emit(code.OpJump, -1)

		// later the pos will change duration backfill
		c.continueContext[l].ips = append(c.continueContext[l].ips, pos)

	case *ast.IfStatement:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		// Emit an `OpJumpNotTruthy` with a bogus value
		jumpNotTruthyPos := c.emit(code.OpJumpIfFalse, -1)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		// Emit an `OpJump` with a bogus value
		jumpPos := c.emit(code.OpJump, -1)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			// don't need this
			// c.emit(code.OpNull)
		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
		}

		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos)
		// since if is a statment now, VM should pop the TOS

		// don't need this
		// c.emit(code.OpPop)

	case *ast.WhileStatement:
		var restart int
		var end int
		c.pushBreakContext()
		c.pushContinueContext()

		defer func() {
			// backfill
			l := len(c.breakContext)
			for _, ip := range c.breakContext[l-1].ips {
				c.changeOperand(ip, end)
			}
			for _, ip := range c.continueContext[l-1].ips {
				c.changeOperand(ip, restart)
			}
			c.popBreakContext()
			c.popContinueContext()
		}()

		restart = len(c.currentInstructions())
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpToEnd := c.emit(code.OpJumpIfFalse, -1)

		err = c.Compile(node.Body)

		if err != nil {
			return err
		}

		c.emit(code.OpJump, restart)
		end = len(c.currentInstructions())
		c.changeOperand(jumpToEnd, end)

		// since while is not a expression, the TOS should be the same
		// before compiling while statement

	case *ast.DoWhileStatement:
		var restart int
		var end int
		var err error

		c.pushBreakContext()
		c.pushContinueContext()

		defer func() {
			// backfill
			l := len(c.breakContext)
			for _, ip := range c.breakContext[l-1].ips {
				c.changeOperand(ip, end)
			}
			for _, ip := range c.continueContext[l-1].ips {
				c.changeOperand(ip, restart)
			}
			c.popBreakContext()
			c.popContinueContext()
		}()

		restart = len(c.currentInstructions())
		err = c.Compile(node.Body)

		if err != nil {
			return err
		}

		err = c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpToEnd := c.emit(code.OpJumpIfFalse, -1)
		c.emit(code.OpJump, restart)

		end = len(c.currentInstructions())
		c.changeOperand(jumpToEnd, end)
		// since while is not a expression, the TOS should be the same
		// before compiling while statement

	case *ast.ForStatement:
		var restart int
		var end int
		var err error

		c.pushBreakContext()
		c.pushContinueContext()

		defer func() {
			// backfill
			l := len(c.breakContext)
			for _, ip := range c.breakContext[l-1].ips {
				c.changeOperand(ip, end)
			}
			for _, ip := range c.continueContext[l-1].ips {
				c.changeOperand(ip, restart)
			}
			c.popBreakContext()
			c.popContinueContext()
		}()

		// add a new scope for it
		c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
		if c.symbolTable.Outer.Outer == nil {
			c.symbolTable.numDefinitions = 0
		} else {
			c.symbolTable.numDefinitions = c.symbolTable.Outer.numDefinitions
		}
		c.symbolTable.block = true

		defer func() {
			c.symbolTable = c.symbolTable.Outer
		}()

		err = c.Compile(node.Init)
		if err != nil {
			return err
		}

		restart = len(c.currentInstructions())
		if node.Condition != nil {
			err = c.Compile(node.Condition)
			if err != nil {
				return err
			}
		} else {
			c.emit(code.OpTrue)
		}

		jumpToEnd := c.emit(code.OpJumpIfFalse, -1)
		err = c.Compile(node.Body)
		if err != nil {
			return err
		}

		err = c.Compile(node.Increment)
		if err != nil {
			return err
		}

		c.emit(code.OpJump, restart)

		end = len(c.currentInstructions())
		c.changeOperand(jumpToEnd, end)

	case *ast.BlockStatement:
		return c.CompileBlockStatement(node, false)

	case *ast.LetStatement:
		symbol := c.symbolTable.Define(node.Name.Value)

		if node.Value != nil {
			err := c.Compile(node.Value)
			if err != nil {
				return err
			}
		} else {
			c.emit(code.OpNull)
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return fmt.Errorf("undefined variable %s", node.Value)
		}

		c.loadSymbol(symbol)

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpArray, len(node.Elements))

	case *ast.HashLiteral:
		keys := []ast.Expression{}
		for k := range node.Pairs {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		for _, k := range keys {
			err := c.Compile(k)
			if err != nil {
				return err
			}
			err = c.Compile(node.Pairs[k])
			if err != nil {
				return err
			}
		}

		c.emit(code.OpHash, len(node.Pairs)*2)

	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Index)
		if err != nil {
			return err
		}

		c.emit(code.OpIndex)

	case *ast.FunctionLiteral:
		c.enterScope()

		if node.Name != "" {
			c.symbolTable.DefineFunctionName(node.Name)
		}

		if node.Alias != "" {
			c.symbolTable.DefineFunctionName(node.Alias)
		}

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value)
		}

		err := c.CompileBlockStatement(node.Body, true)
		if err != nil {
			return err
		}

		// XXX: Chimp force explicit return statement to
		// return the value which is different with Monkey

		if !c.lastInstructionIs(code.OpReturnValue) {
			c.emit(code.OpReturn)
		}

		freeSymbols := c.symbolTable.FreeSymbols
		numLocals := c.symbolTable.numDefinitions
		instructions := c.leaveScope()

		for _, s := range freeSymbols {
			c.loadSymbol(s)
		}

		compiledFn := &object.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Parameters),
		}

		fmt.Printf("%s\n", instructions.String())

		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))

		if node.Alias != "" {
			symbol := c.symbolTable.Define(node.Alias)
			if symbol.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, symbol.Index)
				c.emit(code.OpGetGlobal, symbol.Index)
			} else {
				c.emit(code.OpSetLocal, symbol.Index)
				c.emit(code.OpGetLocal, symbol.Index)
			}
		}

	case *ast.ReturnStatement:
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return err
		}

		c.emit(code.OpReturnValue)

	case *ast.CallExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}

		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpCall, len(node.Arguments))
	}

	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	// desc, _ := code.Lookup(byte(op))
	// fmt.Printf("%s ", desc.Name)
	// for _, i := range operands {
	// 	fmt.Printf("%d ", i)
	// }
	// fmt.Printf("\n")

	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.currentInstructions())
	updatedInstructions := append(c.currentInstructions(), ins...)

	c.scopes[c.scopeIndex].instructions = updatedInstructions

	return posNewInstruction
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}

	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	previous := c.scopes[c.scopeIndex].previousInstruction

	old := c.currentInstructions()
	new := old[:last.Position]

	c.scopes[c.scopeIndex].instructions = new
	c.scopes[c.scopeIndex].lastInstruction = previous
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()

	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.currentInstructions()[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstruction(opPos, newInstruction)
}

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) enterScope() {
	scope := &CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.currentInstructions()

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer

	return instructions
}

func (c *Compiler) currentScope() *CompilationScope {
	return c.scopes[len(c.scopes)-1]
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
	c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))

	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(code.OpGetFree, s.Index)
	case FunctionScope:
		c.emit(code.OpCurrentClosure)
	}
}

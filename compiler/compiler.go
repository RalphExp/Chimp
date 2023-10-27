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

type Compiler struct {
	constants       []object.Object
	symbolTable     *SymbolTable
	scopes          []CompilationScope
	breakContext    [][]int
	continueContext [][]int
	scopeIndex      int
}

func New() *Compiler {
	mainScope := CompilationScope{
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
		scopes:          []CompilationScope{mainScope},
		scopeIndex:      0,
		breakContext:    make([][]int, 0),
		continueContext: make([][]int, 0),
	}
}

func (c *Compiler) pushBreakContext() {
	c.breakContext = append(c.breakContext, []int{})
}

func (c *Compiler) popBreakContext() {
	l := len(c.breakContext)
	c.breakContext = c.breakContext[0 : l-1]
}

func (c *Compiler) pushContinueContext() {
	c.continueContext = append(c.continueContext, []int{})
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
			c.emit(code.OpSetGlobalNoPop, symbol.Index)
		} else {
			c.emit(code.OpSetLocalNoPop, symbol.Index)
		}

	default:
		return fmt.Errorf("invalid left hand side value in assignment")
	}
	return nil
}

func (c *Compiler) CompileBlockStatement(
	node *ast.BlockStatement,
) error {
	// XXX: to implement block scope, we need to save the stack pointer
	// at the beginning, and restore the sp when exiting the block

	c.emit(code.OpSaveSp)

	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
	defer func() {
		c.symbolTable = c.symbolTable.Outer
		c.emit(code.OpRestoreSp)
	}()

	for _, s := range node.Statements {
		err := c.Compile(s)
		if err != nil {
			return err
		}
	}
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

		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpGreaterThan)
			return nil
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
		case ">":
			c.emit(code.OpGreaterThan)
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
		pos := c.emit(code.OpJump, -1)
		l := len(c.breakContext) - 1
		c.breakContext[l] = append(c.breakContext[l], pos)

	case *ast.ContinueStatement:
		if len(c.continueContext) == 0 {
			return fmt.Errorf("no continue context found")
		}
		pos := c.emit(code.OpJump, -1)
		l := len(c.continueContext) - 1
		c.continueContext[l] = append(c.continueContext[l], pos)

	case *ast.IfStatement:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		// Emit an `OpJumpNotTruthy` with a bogus value
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, -1)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		// don't need this since if is statement
		// if c.lastInstructionIs(code.OpPop) {
		// 	c.removeLastPop()
		// }

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

			// don't need this
			// if c.lastInstructionIs(code.OpPop) {
			// 	c.removeLastPop()
			// }
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
			for _, pos := range c.breakContext[l-1] {
				c.changeOperand(pos, end)
			}
			for _, pos := range c.continueContext[l-1] {
				c.changeOperand(pos, restart)
			}
			c.popBreakContext()
			c.popContinueContext()
		}()

		restart = len(c.currentInstructions())
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jmpToEnd := c.emit(code.OpJumpNotTruthy, -1)

		err = c.Compile(node.Body)
		if err != nil {
			return err
		}

		c.emit(code.OpJump, restart)
		end = len(c.currentInstructions())
		c.changeOperand(jmpToEnd, end)

		// since while is not a expression, the TOS should be the same
		// before compiling while statement

	case *ast.DoWhileStatement:
		var restart int
		var end int
		defer func() {
			// backfill
			l := len(c.breakContext)
			for _, pos := range c.breakContext[l-1] {
				c.changeOperand(pos, end)
			}
			for _, pos := range c.continueContext[l-1] {
				c.changeOperand(pos, restart)
			}
			c.popBreakContext()
			c.popContinueContext()
		}()

		restart = len(c.currentInstructions())

		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		err = c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jmpToEnd := c.emit(code.OpJumpNotTruthy, -1)
		c.emit(code.OpJump, restart)

		end = len(c.currentInstructions())
		c.changeOperand(jmpToEnd, end)
		// since while is not a expression, the TOS should be the same
		// before compiling while statement

	case *ast.BlockStatement:
		return c.CompileBlockStatement(node)

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

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value)
		}

		err := c.CompileBlockStatement(node.Body)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.replaceLastPopWithReturn()
		}

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

		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))

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
	scope := CompilationScope{
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

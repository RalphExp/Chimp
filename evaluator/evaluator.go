package evaluator

import (
	"chimp/ast"
	"chimp/object"
	"chimp/parser"
	"fmt"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {

	// Statements
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.BreakStatement:
		return evalBreak(env)

	case *ast.ContinueStatement:
		return evalContinue(env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		return NULL

	// Expressions
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		if parser.IsAssignmentOperator(node.Operator) {
			return evalAssignmentExression(node, env)
		}

		if parser.IsLogicalOperator(node.Operator) {
			return evalLogicalExpression(node, env)
		}

		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(node.Operator, left, right)

	case *ast.IfStatement:
		return evalIfStatement(node, env)

	case *ast.WhileStatement:
		return evalWhileStatement(node, env)

	case *ast.DoWhileStatement:
		return evalDoWhileStatement(node, env)

	case *ast.ForStatement:
		return evalForStatement(node, env)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		fn := &object.Function{Parameters: params, Env: env, Body: body}
		if node.Name != "" {
			env.Set(node.Name, fn)
		}
		return fn

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	case *ast.HashLiteral:
		return evalHashLiteral(node, env)
	}

	return NULL
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

func evalBreak(env *object.Environment) object.Object {
	if !env.HasBreakContext() {
		return &object.Error{Message: "no break context found"}
	}

	return &object.Break{}
}

func evalContinue(env *object.Environment) object.Object {
	if !env.HasContinueContext() {
		return &object.Error{Message: "no continue context found"}
	}
	return &object.Continue{}
}

func evalBlockStatement(
	block *ast.BlockStatement,
	env *object.Environment,
) object.Object {
	var result object.Object = NULL

	extendedEnv := object.NewEnclosedEnvironment(env)

	for _, statement := range block.Statements {
		result = Eval(statement, extendedEnv)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ ||
				rt == object.ERROR_OBJ ||
				rt == object.BREAK_OBJ ||
				rt == object.CONTINUE_OBJ {
				return result
			}
		}
	}
	return result
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	var b bool = isTruthy(right)
	if b {
		return FALSE
	}
	return TRUE
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalAssignmentExression(
	node *ast.InfixExpression,
	env *object.Environment,
) object.Object {

	switch lhs := node.Left.(type) {
	case *ast.Identifier:
		left, e := env.Get(lhs.Value)
		if e == nil {
			return &object.Error{Message: fmt.Sprintf("variable %s not found", lhs.Value)}
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		op := node.Operator[0]
		switch op {
		case '=':
			e.Set(lhs.Value, right)
		default:
			result := evalInfixExpression(string(op), left, right)
			if isError(result) {
				return result
			}
			e.Set(lhs.Value, result)
		}
		left, _ = env.Get(lhs.Value)
		return left

	case *ast.IndexExpression:
		// TODO:
		return newError("Not implemented")

	default:
		return newError("Invalid left hand side value in assignment")
	}
}

func evalLogicalExpression(
	node *ast.InfixExpression,
	env *object.Environment,
) object.Object {

	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	if node.Operator == "&&" {
		if isTruthy(left) {
			return Eval(node.Right, env)
		} else {
			return left
		}
	} else if node.Operator == "||" {
		if isTruthy(left) {
			return left
		} else {
			return Eval(node.Right, env)
		}
	} else {
		panic(fmt.Sprintf("unknow operator: %s\n", node.Operator))
	}
}

func evalIntegerInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newError("divided %d by 0", leftVal)
		}
		return &object.Integer{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newError("divided %d by 0", leftVal)
		}
		return &object.Integer{Value: leftVal % rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value
	return &object.String{Value: leftVal + rightVal}
}

func evalIfStatement(
	ifs *ast.IfStatement,
	env *object.Environment,
) object.Object {
	condition := Eval(ifs.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ifs.Consequence, env)
	} else if ifs.Alternative != nil {
		return Eval(ifs.Alternative, env)
	}
	return NULL
}

func evalWhileStatement(ws *ast.WhileStatement,
	env *object.Environment,
) object.Object {
	env.PushBreakContext()
	env.PushContinueContext()

	defer func() {
		env.PopBreakContext()
		env.PopContinueContext()
	}()

	for {
		condition := Eval(ws.Condition, env)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			break
		}

		obj := Eval(ws.Body, env)
		if isError(obj) {
			return obj
		}

		if obj == nil {
			continue
		} else if obj.Type() == object.BREAK_OBJ {
			break
		} else if obj.Type() == object.CONTINUE_OBJ {
			continue
		} else if obj.Type() == object.RETURN_VALUE_OBJ {
			return obj
		}
	}
	return NULL
}

func evalDoWhileStatement(
	dw *ast.DoWhileStatement,
	env *object.Environment,
) object.Object {
	env.PushBreakContext()
	env.PushContinueContext()

	defer func() {
		env.PopBreakContext()
		env.PopContinueContext()
	}()

	for {
		obj := Eval(dw.Body, env)
		if isError(obj) {
			return obj
		}

		if obj != nil {
			if obj.Type() == object.BREAK_OBJ {
				break
			} else if obj.Type() == object.CONTINUE_OBJ {
				continue
			} else if obj.Type() == object.RETURN_VALUE_OBJ {
				return obj
			}
		}

		condition := Eval(dw.Condition, env)
		if isError(condition) {
			return condition
		}
		if isTruthy(condition) {
			continue
		}
		break
	}

	return NULL
}

func evalForStatement(
	f *ast.ForStatement,
	env *object.Environment,
) object.Object {

	env.PushBreakContext()
	env.PushContinueContext()

	defer func() {
		env.PopBreakContext()
		env.PopContinueContext()
	}()

	// for statment declares variable(s) in its own env
	forEnv := object.NewEnclosedEnvironment(env)
	init := Eval(f.Init, forEnv)
	if isError(init) {
		return init
	}

	for {
		if f.Condition != nil {
			condition := Eval(f.Condition, forEnv)
			if isError(condition) {
				return condition
			}
			if !isTruthy(condition) {
				break
			}
		}

		obj := Eval(f.Body, forEnv)
		if isError(obj) {
			return obj
		}

		if obj.Type() == object.BREAK_OBJ {
			break
		} else if obj.Type() == object.CONTINUE_OBJ {
			continue
		} else if obj.Type() == object.RETURN_VALUE_OBJ {
			return obj
		}

		incr := Eval(f.Increment, forEnv)
		if isError(incr) {
			return incr
		}
	}
	return NULL
}

func evalIdentifier(
	node *ast.Identifier,
	env *object.Environment,
) object.Object {
	if val, e := env.Get(node.Value); e != nil {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: " + node.Value)
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {

	case *object.Boolean:
		return obj.Value

	case *object.Null:
		return false

	case *object.Integer:
		return obj.Value != 0

	case *object.String:
		return len(obj.Value) > 0

	default:
		return true
	}
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func evalExpressions(
	exps []ast.Expression,
	env *object.Environment,
) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch fn := fn.(type) {

	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *object.Builtin:
		if result := fn.Fn(args...); result != nil {
			return result
		}
		return NULL

	default:
		return newError("not a function: %s", fn.Type())
	}
}

func extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObject.Elements[idx]
}

func evalHashLiteral(
	node *ast.HashLiteral,
	env *object.Environment,
) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", key.Type())
		}

		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}
}

func evalHashIndexExpression(hash, index object.Object) object.Object {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return NULL
	}

	return pair.Value
}

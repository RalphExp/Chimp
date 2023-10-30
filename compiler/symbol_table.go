package compiler

type SymbolScope string

const (
	LocalScope    SymbolScope = "LOCAL"
	GlobalScope   SymbolScope = "GLOBAL"
	BuiltinScope  SymbolScope = "BUILTIN"
	FreeScope     SymbolScope = "FREE"
	FunctionScope SymbolScope = "FUNCTION"
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	Outer *SymbolTable

	store          map[string]Symbol
	numDefinitions int
	block          bool // introduced by block

	FreeSymbols []Symbol
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	free := []Symbol{}
	return &SymbolTable{store: s, FreeSymbols: free}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}

	s.store[name] = symbol
	s.numDefinitions++
	return symbol
}

// func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
// 	obj, ok := s.store[name]
// 	if !ok && s.Outer != nil {
// 		obj, ok = s.Outer.Resolve(name)
// 		if !ok {
// 			return obj, ok
// 		}

// 		if obj.Scope == GlobalScope || obj.Scope == BuiltinScope {
// 			return obj, ok
// 		}

// 		free := s.defineFree(obj)
// 		return free, true
// 	}
// 	return obj, ok
// }

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	inBlock := s.block
	obj, ok := s.store[name]
	if ok {
		return obj, ok
	}
	s = s.Outer
	if s == nil {
		return obj, ok
	}

	for {
		obj, ok := s.store[name]
		if ok {
			if obj.Scope == GlobalScope ||
				obj.Scope == BuiltinScope ||
				obj.Scope == FunctionScope {
				return obj, ok
			}
			if !inBlock {
				free := s.defineFree(obj)
				return free, true
			}
			// Local Scope
			return obj, true
		}
		inBlock = inBlock && s.block
		s = s.Outer
		if s == nil {
			return obj, ok
		}
	}
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Index: index, Scope: BuiltinScope}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) DefineFunctionName(name string) Symbol {
	symbol := Symbol{Name: name, Index: 0, Scope: FunctionScope}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) defineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)

	symbol := Symbol{Name: original.Name, Index: len(s.FreeSymbols) - 1}
	symbol.Scope = FreeScope

	s.store[original.Name] = symbol
	return symbol
}

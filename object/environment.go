package object

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

const (
	BreakState = 1 << iota
	ContinueState
	ErrorState
)

type Environment struct {
	store      map[string]Object
	outer      *Environment
	brkContext int // break context
	cntContext int // continue context
	retContext int // return context
}

func (e *Environment) PushBreakContext() {
	e.brkContext++
}

func (e *Environment) PushContinueContext() {
	e.cntContext++
}

func (e *Environment) PopBreakContext() {
	e.brkContext--
}

func (e *Environment) PopContinueContext() {
	e.cntContext--
}

func (e *Environment) HasBreakContext() bool {
	for e != nil {
		if e.brkContext > 0 {
			return true
		}
		e = e.outer
	}
	return false
}

func (e *Environment) HasContinueContext() bool {
	for e != nil {
		if e.cntContext > 0 {
			return true
		}
		e = e.outer
	}
	return false
}

func (e *Environment) Get(name string) (Object, *Environment) {
	for e != nil {
		obj, ok := e.store[name]
		if ok {
			return obj, e
		} else if e.outer != nil {
			e = e.outer
		} else {
			break
		}
	}
	return nil, nil
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

package vm

import (
	"chimp/code"
	"chimp/object"
)

type Frame struct {
	cl          *object.Closure
	ip          int
	basePointer int
	blocks      []int // saving stack pointer for the given frame
}

func NewFrame(cl *object.Closure, basePointer int) *Frame {
	f := &Frame{
		cl:          cl,          // closure object
		ip:          -1,          // instruction pointer
		basePointer: basePointer, // base pointer points to callee(the current function) + 1
		blocks:      []int{basePointer},
	}

	return f
}

func (f *Frame) Instructions() code.Instructions {
	return f.cl.Fn.Instructions
}

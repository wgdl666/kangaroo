package bizerror

import (
	"bytes"
	"fmt"
	"runtime"
)

func stack() []byte {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	stack := pcs[0:n]

	var str bytes.Buffer
	str.Grow(64 * len(stack))
	for _, pc := range stack {
		frame := frame(pc)
		str.WriteString(fmt.Sprintf("%s:%d %s\n", frame.file(), frame.line(), frame.name()))
	}
	return str.Bytes()
}

type frame uintptr

// pc returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f frame) pc() uintptr { return uintptr(f) - 1 }

// file returns the full path to the file that contains the
// function for this frame's pc.
func (f frame) file() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	file, _ := fn.FileLine(f.pc())
	return file
}

// line returns the line number of source code of the
// function for this frame's pc.
func (f frame) line() int {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}

func (f frame) name() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}

	return fn.Name()
}

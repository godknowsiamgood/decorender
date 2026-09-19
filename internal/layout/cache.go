package layout

import (
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type Cache struct {
	// programs is keyed by the expression source itself. A hash would risk two
	// different expressions sharing a compiled program on collision.
	programs   map[string]*vm.Program
	programsMx sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		programs: make(map[string]*vm.Program),
	}
}

// program returns the compiled form of src, compiling it on first use.
//
// The lock is only held around the map itself, never across evaluation:
// vm.Program is read-only once compiled and expr.Run builds a fresh VM per
// call, so running an expression needs no lock at all. Holding it across Run
// would serialise every concurrent render on a single mutex.
func (c *Cache) program(src string) (*vm.Program, error) {
	c.programsMx.RLock()
	program := c.programs[src]
	c.programsMx.RUnlock()

	if program != nil {
		return program, nil
	}

	program, err := expr.Compile(src)
	if err != nil {
		return nil, err
	}

	c.programsMx.Lock()
	if existing, ok := c.programs[src]; ok {
		program = existing // another goroutine compiled it first
	} else {
		c.programs[src] = program
	}
	c.programsMx.Unlock()

	return program, nil
}

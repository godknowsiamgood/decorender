package layout

import (
	"github.com/antonmedv/expr/vm"
	"sync"
)

type Cache struct {
	programs   map[uint]*vm.Program
	programsMx sync.Mutex
}

func NewCache() *Cache {
	return &Cache{
		programs: make(map[uint]*vm.Program),
	}
}

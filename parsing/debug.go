package parsing

func KeepDebugNodes(n Node) Node {
	var debugNode *Node
	iterateNode(n, func(n Node) bool {
		if n.DebugOnly != "" {
			debugNode = &n
			return false
		}
		return true
	})
	if debugNode != nil {
		n.Inner = []Node{*debugNode}
	}
	return n
}

func iterateNode(n Node, cb func(n Node) bool) {
	if cb(n) == false {
		return
	}
	for _, cn := range n.Inner {
		iterateNode(cn, cb)
	}
}

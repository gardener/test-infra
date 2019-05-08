package node

func (list List) AddParents(parents ...*Node) {
	for _, node := range list {
		node.AddParents(parents...)
	}
}
func (list List) RemoveParents(parents ...*Node) {
	for _, node := range list {
		for _, parent := range parents {
			node.RemoveParent(parent)
		}
	}
}
func (list List) ClearParents() {
	for _, node := range list {
		node.ClearParents()
	}
}

func (list List) AddChildren(children ...*Node) {
	for _, node := range list {
		node.AddChildren(children...)
	}
}
func (list List) RemoveChildren(children ...*Node) {
	for _, node := range list {
		for _, child := range children {
			node.RemoveChild(child)
		}
	}
}
func (list List) ClearChildren() {
	for _, node := range list {
		node.ClearChildren()
	}
}

// GetParents returns a set of all parents
func (list List) GetChildren() List {
	children := make(List, 0)
	childrenSet := make(map[*Node]bool)
	for _, node := range list {
		for _, child := range node.Children {
			if _, ok := childrenSet[child]; !ok {
				children = append(children, child)
				childrenSet[child] = true
			}
		}
	}
	return children
}

// GetParents returns a set of all parents
func (list List) GetParents() List {
	parents := make(List, 0)
	parentsSet := make(map[*Node]bool)
	for _, node := range list {
		for _, parent := range node.Parents {
			if _, ok := parentsSet[parent]; !ok {
				parents = append(parents, parent)
				parentsSet[parent] = true
			}
		}
	}
	return parents
}

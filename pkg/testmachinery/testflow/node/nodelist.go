package node

import "sort"

// NewSet creates a new set of nodes
func NewSet(nodes ...*Node) Set {
	set := make(Set, 0)
	set.Add(nodes...)
	return set
}

// Copy creates a deep copy of the set
func (s Set) Copy() Set {
	newSet := make(Set, len(s))
	for key, val := range s {
		newSet[key] = val
	}
	return newSet
}

func (s Set) Has(n *Node) bool {
	_, ok := s[n]
	return ok
}

// Add adds nodes to the set
func (s Set) Add(nodes ...*Node) {
	for _, n := range nodes {
		s[n] = empty{}
	}
}

// Removes nodes form the set
func (s Set) Remove(nodes ...*Node) {
	for _, n := range nodes {
		delete(s, n)
	}
}

// AddParents adds nodes as parents.
func (s Set) AddParents(parents ...*Node) {
	for node := range s {
		node.AddParents(parents...)
	}
}

// RemoveParents removes nodes from parents.
func (s Set) RemoveParents(parents ...*Node) {
	for n := range s {
		for parent := range n.Parents {
			n.RemoveParent(parent)
		}
	}
}

// Clear Parents removes all parents.
func (s Set) ClearParents() {
	for node := range s {
		node.ClearParents()
	}
}

// AddChildren adds nodes as children.
func (s Set) AddChildren(children ...*Node) {
	for node := range s {
		node.AddChildren(children...)
	}
}

// RemoveChildren removes nodes from children
func (s Set) RemoveChildren(children ...*Node) {
	for n := range s {
		for child := range n.Children {
			n.RemoveChild(child)
		}
	}
}

// ClearChildren removes all children.
func (s Set) ClearChildren() {
	for node := range s {
		node.ClearChildren()
	}
}

// GetParents returns a set of all parents
func (s Set) GetChildren() Set {
	children := make(Set)
	for node := range s {
		for child := range node.Children {
			if _, ok := children[child]; !ok {
				children.Add(child)
			}
		}
	}
	return children
}

// GetParents returns a set of all parents
func (s Set) GetParents() Set {
	parents := make(Set, 0)
	for node := range s {
		for parent := range node.Parents {
			if _, ok := parents[parent]; !ok {
				parents.Add(parent)
			}
		}
	}
	return parents
}

type sortableSliceOfNodes []*Node

func (s sortableSliceOfNodes) Len() int           { return len(s) }
func (s sortableSliceOfNodes) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
func (s sortableSliceOfNodes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Set return all nodes of the set as an array
func (s Set) List() []*Node {
	res := make(sortableSliceOfNodes, 0, len(s))
	for key := range s {
		res = append(res, key)
	}
	sort.Sort(res)
	return []*Node(res)
}

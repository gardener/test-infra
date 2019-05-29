package node

import "sort"

func NewSet(nodes ...*Node) Set {
	set := make(Set, 0)
	set.Add(nodes...)
	return set
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

func (s Set) AddParents(parents ...*Node) {
	for node := range s {
		node.AddParents(parents...)
	}
}
func (s Set) RemoveParents(parents ...*Node) {
	for n := range s {
		for parent := range n.Parents {
			n.RemoveParent(parent)
		}
	}
}
func (s Set) ClearParents() {
	for node := range s {
		node.ClearParents()
	}
}

func (s Set) AddChildren(children ...*Node) {
	for node := range s {
		node.AddChildren(children...)
	}
}
func (s Set) RemoveChildren(children ...*Node) {
	for n := range s {
		for child := range n.Children {
			n.RemoveChild(child)
		}
	}
}
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

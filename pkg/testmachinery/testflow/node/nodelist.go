package node

import "k8s.io/apimachinery/pkg/util/sets"

// NewSet creates a new set of nodes
func NewSet(nodes ...*Node) *Set {
	set := Set{
		set: make(map[*Node]sets.Empty),
	}
	set.Add(nodes...)
	return &set
}

// New creates a deep copy of the set
func (s *Set) Copy() *Set {
	newIntSet := make(map[*Node]sets.Empty, len(s.set))
	for key, val := range s.set {
		newIntSet[key] = val
	}
	set := &Set{
		set: newIntSet,
	}
	set.Add(s.List()...)
	return set
}

// is a internal iteration through the list items
func (s *Set) iterateList() chan *listItem {
	c := make(chan *listItem)
	go func() {
		n := s.listStart
		for n != nil {
			c <- n
			n = n.next
		}
		close(c)
	}()
	return c
}

// returns the list item for a node
func (s *Set) getListItem(node *Node) *listItem {
	for item := range s.iterateList() {
		if item.node == node {
			return item
		}
	}
	return nil
}

func (s *Set) Iterate() chan *Node {
	c := make(chan *Node)
	go func() {
		n := s.listStart
		for n != nil {
			c <- n.node
			n = n.next
		}
		close(c)
	}()
	return c
}

func (s *Set) IterateInverse() chan *Node {
	c := make(chan *Node)
	go func() {
		n := s.listEnd
		for n != nil {
			c <- n.node
			n = n.previous
		}
		close(c)
	}()
	return c
}

// Len returns the length of the list
func (s *Set) Len() int {
	return len(s.set)
}

// Has returns true if the given node is in the list
func (s *Set) Has(n *Node) bool {
	_, ok := s.set[n]
	return ok
}

// Last returns the last node of the list
func (s *Set) Last() *Node {
	if s.listEnd == nil {
		return nil
	}
	return s.listEnd.node
}

// Add adds nodes to the set
func (s *Set) Add(nodes ...*Node) *Set {
	for _, n := range nodes {
		if _, ok := s.set[n]; ok {
			continue
		}
		s.set[n] = sets.Empty{}
		newItem := &listItem{
			node:     n,
			previous: s.listEnd,
		}
		if s.listEnd != nil {
			newItem.previous.next = newItem
		}
		if s.listStart == nil {
			s.listStart = newItem
		}
		s.listEnd = newItem
	}
	return s
}

// Removes nodes form the set
func (s *Set) Remove(nodes ...*Node) *Set {
	for _, n := range nodes {
		if _, ok := s.set[n]; !ok {
			continue
		}
		delete(s.set, n)
		item := s.getListItem(n)
		if item == nil {
			// this should never happen as we will have a inconsistency in the list
			// but for now we expect that the item is then already deleted
			continue
		}
		if item.previous != nil {
			item.previous.next = item.next
		} else {
			s.listStart = item.next
		}
		if item.next != nil {
			item.next.previous = item.previous
		} else {
			s.listEnd = item.previous
		}
		// let go garbage collect the item
	}
	return s
}

// AddParents adds nodes as parents.
func (s *Set) AddParents(parents ...*Node) *Set {
	for node := range s.Iterate() {
		node.AddParents(parents...)
	}
	return s
}

// RemoveParents removes nodes from parents.
func (s *Set) RemoveParents(parents ...*Node) *Set {
	for n := range s.Iterate() {
		for _, parent := range parents {
			n.RemoveParent(parent)
		}
	}
	return s
}

// Clear Parents removes all parents.
func (s *Set) ClearParents() *Set {
	for node := range s.Iterate() {
		node.ClearParents()
	}
	return s
}

// AddChildren adds nodes as children.
func (s *Set) AddChildren(children ...*Node) *Set {
	for node := range s.Iterate() {
		node.AddChildren(children...)
	}
	return s
}

// RemoveChildren removes nodes from children
func (s *Set) RemoveChildren(children ...*Node) {
	for n := range s.Iterate() {
		for _, child := range children {
			n.RemoveChild(child)
		}
	}
}

// ClearChildren removes all children.
func (s *Set) ClearChildren() *Set {
	for node := range s.Iterate() {
		node.ClearChildren()
	}
	return s
}

// GetChildren returns a set of all children
func (s *Set) GetChildren() *Set {
	children := NewSet()
	for node := range s.Iterate() {
		for child := range node.Children.Iterate() {
			if !children.Has(child) {
				children.Add(child)
			}
		}
	}
	return children
}

// GetParents returns a set of all parents
func (s *Set) GetParents() *Set {
	parents := NewSet()
	for node := range s.Iterate() {
		for parent := range node.Parents.Iterate() {
			if !parents.Has(parent) {
				parents.Add(parent)
			}
		}
	}
	return parents
}

// List return all nodes of the set as an array
func (s Set) List() []*Node {
	var (
		list = make([]*Node, len(s.set))
		i    = 0
	)
	for node := range s.Iterate() {
		list[i] = node
		i++
	}
	return list
}

// Set returns map of the set
func (s Set) Set() map[*Node]sets.Empty {
	return s.set
}

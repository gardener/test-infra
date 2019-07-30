package node

// NewSet creates a new set of nodes
func NewSet(nodes ...*Node) *Set {
	set := Set{
		set:  make(map[*Node]int, 0),
		list: make([]*Node, 0),
	}
	set.Add(nodes...)
	return &set
}

// Copy creates a deep copy of the set
func (s *Set) Copy() *Set {
	newSet := make(map[*Node]int, len(s.set))
	for key, val := range s.set {
		newSet[key] = val
	}
	newList := make([]*Node, len(s.list))
	for i, v := range s.list {
		newList[i] = v
	}

	return &Set{
		set:  newSet,
		list: newList,
	}
}

func (s *Set) Iterate() chan *Node {
	c := make(chan *Node)
	go func() {
		for _, n := range s.list {
			c <- n
		}
		close(c)
	}()
	return c
}

func (s *Set) IterateInverse() chan *Node {
	c := make(chan *Node)
	go func() {
		for i := len(s.list) - 1; i >= 0; i-- {
			c <- s.list[i]
		}
		close(c)
	}()
	return c
}

func (s *Set) Len() int {
	return len(s.set)
}

func (s *Set) Has(n *Node) bool {
	_, ok := s.set[n]
	return ok
}

// Add adds nodes to the set
func (s *Set) Add(nodes ...*Node) {
	for _, n := range nodes {
		s.set[n] = len(s.list)
		s.list = append(s.list, n)
	}
}

// Removes nodes form the set
func (s *Set) Remove(nodes ...*Node) {
	for _, n := range nodes {
		i := s.set[n]
		delete(s.set, n)
		s.list = append(s.list[:i], s.list[i+1:]...)
	}
}

// AddParents adds nodes as parents.
func (s *Set) AddParents(parents ...*Node) {
	for node := range s.Iterate() {
		node.AddParents(parents...)
	}
}

// RemoveParents removes nodes from parents.
func (s *Set) RemoveParents(parents ...*Node) {
	for n := range s.Iterate() {
		for parent := range n.Parents.set {
			n.RemoveParent(parent)
		}
	}
}

// Clear Parents removes all parents.
func (s *Set) ClearParents() {
	for node := range s.Iterate() {
		node.ClearParents()
	}
}

// AddChildren adds nodes as children.
func (s *Set) AddChildren(children ...*Node) {
	for node := range s.Iterate() {
		node.AddChildren(children...)
	}
}

// RemoveChildren removes nodes from children
func (s *Set) RemoveChildren(children ...*Node) {
	for n := range s.Iterate() {
		for child := range n.Children.Iterate() {
			n.RemoveChild(child)
		}
	}
}

// ClearChildren removes all children.
func (s *Set) ClearChildren() {
	for node := range s.Iterate() {
		node.ClearChildren()
	}
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
	return s.list
}

// Set returns map of the set
func (s Set) Set() map[*Node]int {
	return s.set
}

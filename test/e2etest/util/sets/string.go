package sets

import (
	"sort"
	"strings"
)

type Empty struct{}

// sets.StringSet is a set of strings, implemented via map[string]struct{} for minimal memory consumption.
type StringSet map[string]Empty

// NewStringSet creates a StringSet from a list of values.
func NewStringSet(items ...string) StringSet {
	ss := StringSet{}
	ss.Insert(items...)
	return ss
}

// Insert adds items to the set.
func (s StringSet) Insert(items ...string) {
	for _, item := range items {
		s[item] = Empty{}
	}
}

func (s StringSet) GetSetOfMatching(substrings StringSet) StringSet {
	matchedItems := NewStringSet()
	for substring := range substrings {
		for mapItem := range s {
			if strings.Contains(mapItem, substring) {
				matchedItems.Insert(mapItem)
			}
		}
	}
	return matchedItems
}

func (s StringSet) GetMatching(substring string) string {
	for mapItem := range s {
		if strings.Contains(mapItem, substring) {
			return mapItem
		}
	}
	return ""
}

func (s StringSet) DeleteMatching(items ...string) {
	for _, item := range items {
		s.Delete(s.GetMatching(item))
	}
}

// Delete removes all items from the set.
func (s StringSet) Delete(items ...string) {
	for _, item := range items {
		delete(s, item)
	}
}

// Union returns a new set which includes items in either s1 or s2.
// For example:
// s1 = {a1, a2}
// s2 = {a3, a4}
// s1.Union(s2) = {a1, a2, a3, a4}
// s2.Union(s1) = {a1, a2, a3, a4}
func (s1 StringSet) Union(s2 StringSet) StringSet {
	result := NewStringSet()
	for key := range s1 {
		result.Insert(key)
	}
	for key := range s2 {
		result.Insert(key)
	}
	return result
}

type sortableSliceOfString []string

func (s sortableSliceOfString) Len() int           { return len(s) }
func (s sortableSliceOfString) Less(i, j int) bool { return lessString(s[i], s[j]) }
func (s sortableSliceOfString) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// List returns the contents as a sorted string slice.
func (s StringSet) List() []string {
	res := make(sortableSliceOfString, 0, len(s))
	for key := range s {
		res = append(res, key)
	}
	sort.Sort(res)
	return []string(res)
}

// UnsortedList returns the slice with contents in random order.
func (s StringSet) UnsortedList() []string {
	res := make([]string, 0, len(s))
	for key := range s {
		res = append(res, key)
	}
	return res
}

// Returns a single element from the set.
func (s StringSet) PopAny() (string, bool) {
	for key := range s {
		s.Delete(key)
		return key, true
	}
	var zeroValue string
	return zeroValue, false
}

// Len returns the size of the set.
func (s StringSet) Len() int {
	return len(s)
}

func lessString(lhs, rhs string) bool {
	return lhs < rhs
}

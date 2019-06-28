package sets

import (
	"fmt"
	"log"
	"os"
	"regexp"

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

func (s StringSet) GetMatchingOfSet(needles StringSet) StringSet {
	matchedItems := NewStringSet()
	for hayItem := range s {
		for needle := range needles {
			if strings.Contains(hayItem, needle) {
				matchedItems.Insert(hayItem)
			}
		}
	}
	return matchedItems
}

func (s StringSet) GetMatching(substring string) StringSet {
	matches := NewStringSet()
	for mapItem := range s {
		if strings.Contains(mapItem, substring) {
			matches.Insert(mapItem)
		}
	}
	return matches
}

func (s StringSet) GetMatchingForTestcase(testcaseName, skip, focus string) StringSet {
	matches := NewStringSet()
	var skipRegex *regexp.Regexp
	var focusRegex *regexp.Regexp
	if skip != "" {
		skipRegex = regexp.MustCompile(skip)
	}
	if focus != "" {
		focusRegex = regexp.MustCompile(focus)
	}
	for mapItem := range s {
		if strings.Contains(mapItem, testcaseName) {
			if skip == "" && focus == "" {
				matches.Insert(mapItem)
				continue
			}
			if skipRegex != nil && !skipRegex.MatchString(mapItem) {
				matches.Insert(mapItem)
				continue
			}
			if focusRegex != nil && focusRegex.MatchString(mapItem) {
				matches.Insert(mapItem)
				continue
			}
		}
	}
	return matches
}

func (s StringSet) DeleteMatchingSet(needles StringSet) {
	for needle := range needles {
		for match := range s.GetMatching(needle) {
			s.Delete(match)
		}
	}
}

func (s StringSet) DeleteMatching(items ...string) {
	for _, item := range items {
		for match := range s.GetMatching(item) {
			s.Delete(match)
		}
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

// Difference returns a set of objects that are not in s2
// For example:
// s1 = {a1, a2, a3}
// s2 = {a1, a2, a4, a5}
// s1.Difference(s2) = {a3}
// s2.Difference(s1) = {a4, a5}
func (s StringSet) Difference(s2 StringSet) StringSet {
	result := NewStringSet()
	for key := range s {
		if !s2.Has(key) {
			result.Insert(key)
		}
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

// Has returns true if and only if item is contained in the set.
func (s StringSet) Has(item string) bool {
	_, contained := s[item]
	return contained
}

// Len returns the size of the set.
func (s StringSet) Len() int {
	return len(s)
}

func (s StringSet) WriteToFile(filepath string) {
	file, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}
	for item := range s {
		fmt.Fprintln(file, item)
	}
}

func lessString(lhs, rhs string) bool {
	return lhs < rhs
}

package credentialhandler

import (
	"errors"
	"slices"
)

// ArrayMap is a map of string to string, backed by a simple two dimensional
// array. Keys are preserved in insertion order.
//
// It is only suited to small maps, as its lookup and set operations are O(n).
//
// This is not to be used for performance, only for guaranteeing consistent
// iteration ordering.
type ArrayMap struct {
	s [][]string
}

func NewMap(cap int) *ArrayMap {
	return &ArrayMap{
		s: make([][]string, 0, cap),
	}
}

func NewMapFromArray(a [][]string) *ArrayMap {
	sm := NewMap(len(a))
	sm.AddAll(a)
	return sm
}

func (m *ArrayMap) Len() int {
	return len(m.s)
}

func (m *ArrayMap) AddAll(a [][]string) {
	m.s = slices.Grow(m.s, len(a))

	for _, kv := range a {
		m.Set(kv[0], kv[1])
	}
}

func (m *ArrayMap) Set(k, v string) {
	for i := range m.s {
		if m.s[i][0] == k {
			m.s[i][1] = v
			return
		}
	}

	// not found, append
	m.s = append(m.s, []string{k, v})
}

func (m *ArrayMap) Get(k string) string {
	v, _ := m.Lookup(k)
	return v
}

func (m *ArrayMap) Lookup(k string) (string, bool) {
	for i := range m.s {
		if m.s[i][0] == k {
			return m.s[i][1], true
		}
	}

	return "", false
}

// Array returns a deep copy of the stored elements as a two-dimensional
// string array.
func (m *ArrayMap) Array() [][]string {
	n := slices.Clone(m.s)
	for i := range n {
		n[i] = slices.Clone(n[i])
	}
	return n
}

func (m *ArrayMap) Iter() iterator {
	return iterator{m, 0}
}

type iterator struct {
	m *ArrayMap
	i int
}

func (i *iterator) Next() (string, string) {
	if i.i >= i.m.Len() {
		panic(errors.New("attempted to iterate past the end of the map"))
	}

	k, v := i.m.s[i.i][0], i.m.s[i.i][1]
	i.i++

	return k, v
}

func (i *iterator) HasNext() bool {
	return i.i < i.m.Len()
}

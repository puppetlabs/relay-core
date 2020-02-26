package graph

import (
	"github.com/puppetlabs/horsehead/v2/datastructure"
)

type unenforcedSliceEdgeSet []Edge

func (es unenforcedSliceEdgeSet) Contains(edge Edge) bool {
	for _, e := range es {
		if e == edge {
			return true
		}
	}

	return false
}

func (es unenforcedSliceEdgeSet) Count() uint {
	return uint(len(es))
}

func (es unenforcedSliceEdgeSet) AsSlice() []Edge {
	return es
}

func (es unenforcedSliceEdgeSet) ForEach(fn EdgeSetIterationFunc) error {
	for _, edge := range es {
		if err := fn(edge); err != nil {
			return err
		}
	}

	return nil
}

func (es *unenforcedSliceEdgeSet) Add(edge Edge) {
	*es = append(*es, edge)
}

type edgeSet struct {
	storage datastructure.Set
}

func (es *edgeSet) Contains(edge Edge) bool {
	return es.storage.Contains(edge)
}

func (es *edgeSet) Count() uint {
	return uint(es.storage.Size())
}

func (es *edgeSet) AsSlice() []Edge {
	s := make([]Edge, es.Count())

	i := 0
	es.ForEach(func(edge Edge) error {
		s[i] = edge
		i++

		return nil
	})

	return s
}

func (es *edgeSet) ForEach(fn EdgeSetIterationFunc) error {
	return es.storage.ForEachInto(fn)
}

func (es *edgeSet) Add(edge Edge) {
	es.storage.Add(edge)
}

func (es *edgeSet) Remove(edge Edge) {
	es.storage.Remove(edge)
}

// NewMutableEdgeSet creates a new mutable edge set using the given underlying
// set to store the edges.
func NewMutableEdgeSet(storage datastructure.Set) MutableEdgeSet {
	return &edgeSet{storage}
}

// Portions of this file are derived from JGraphT, a free Java graph-theory
// library.
//
// (C) Copyright 2003-2018, by Liviu Rau and Contributors.

package traverse

import (
	"reflect"

	"github.com/puppetlabs/horsehead/v2/graph"
)

var depthFirstSentinel = struct{}{}

type depthFirstStackElement struct {
	vertex graph.Vertex

	prevStackElement *depthFirstStackElement
}

type DepthFirstTraverser struct {
	g     graph.Graph
	start graph.Vertex
}

func (t *DepthFirstTraverser) forEachEdgeOf(vertex graph.Vertex, fn graph.EdgeSetIterationFunc) {
	var edges graph.EdgeSet
	if dg, ok := t.g.(graph.DirectedGraph); ok {
		edges, _ = dg.OutgoingEdgesOf(vertex)
	} else {
		edges, _ = t.g.EdgesOf(vertex)
	}

	edges.ForEach(fn)
}

func (t *DepthFirstTraverser) ForEach(fn func(vertex graph.Vertex) error) error {
	seen := make(map[graph.Vertex]struct{})

	stack := &depthFirstStackElement{vertex: t.start}
	for stack != nil {
		var cur graph.Vertex
		cur, stack = stack.vertex, stack.prevStackElement

		if _, found := seen[cur]; found {
			continue
		}

		seen[cur] = depthFirstSentinel

		if err := fn(cur); err != nil {
			return err
		}

		t.forEachEdgeOf(cur, func(edge graph.Edge) error {
			next, _ := graph.OppositeVertexOf(t.g, edge, cur)

			stack = &depthFirstStackElement{
				vertex:           next,
				prevStackElement: stack,
			}

			return nil
		})
	}

	return nil
}

func (t *DepthFirstTraverser) ForEachInto(fn interface{}) error {
	fnr := reflect.ValueOf(fn)
	fnt := fnr.Type()

	if fnt.NumOut() != 1 {
		panic(ErrInvalidFuncSignature)
	}

	return t.ForEach(func(vertex graph.Vertex) error {
		p := reflect.ValueOf(vertex)
		if !p.IsValid() {
			p = reflect.Zero(fnt.In(0))
		}

		r := fnr.Call([]reflect.Value{p})

		err := r[0]
		if !err.IsNil() {
			return err.Interface().(error)
		}

		return nil
	})
}

func NewDepthFirstTraverser(g graph.Graph, start graph.Vertex) *DepthFirstTraverser {
	return &DepthFirstTraverser{
		g:     g,
		start: start,
	}
}

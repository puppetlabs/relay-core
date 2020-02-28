// Portions of this file are derived from JGraphT, a free Java graph-theory
// library.
//
// (C) Copyright 2004-2017, by Marden Neubert and Contributors.

package traverse

import (
	"reflect"

	"github.com/puppetlabs/horsehead/v2/graph"
)

type TopologicalOrderTraverser struct {
	g graph.DirectedGraph
}

func (t *TopologicalOrderTraverser) Vertices() ([]graph.Vertex, error) {
	vertices := make([]graph.Vertex, t.g.Vertices().Count())

	i := 0
	err := t.ForEach(func(vertex graph.Vertex) error {
		vertices[i] = vertex
		i++

		return nil
	})
	if err != nil {
		return nil, err
	}

	return vertices, nil
}

func (t *TopologicalOrderTraverser) ForEach(fn func(vertex graph.Vertex) error) error {
	if t.g.Vertices().Count() == 0 {
		return nil
	}

	var queue []graph.Vertex
	remaining := make(map[graph.Vertex]uint)

	// Find our starting point(s).
	t.g.Vertices().ForEach(func(vertex graph.Vertex) error {
		if in, _ := t.g.InDegreeOf(vertex); in == 0 {
			queue = append(queue, vertex)
		}

		remaining[vertex], _ = t.g.InDegreeOf(vertex)

		return nil
	})

	if len(queue) == 0 {
		return ErrCyclicGraph
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if err := fn(cur); err != nil {
			return err
		}

		edges, _ := t.g.OutgoingEdgesOf(cur)
		edges.ForEach(func(edge graph.Edge) error {
			next, _ := t.g.TargetVertexOf(edge)

			if remaining[next] > 0 {
				remaining[next]--

				if remaining[next] == 0 {
					queue = append(queue, next)
				}
			}

			return nil
		})
	}

	return nil
}

func (t *TopologicalOrderTraverser) ForEachInto(fn interface{}) error {
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

func NewTopologicalOrderTraverser(g graph.DirectedGraph) *TopologicalOrderTraverser {
	return &TopologicalOrderTraverser{g}
}

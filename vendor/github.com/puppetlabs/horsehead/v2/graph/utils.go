package graph

import (
	"sync/atomic"
)

var defaultEdgeIndex uint64

type defaultEdge struct {
	index uint64
}

// NewEdge creates a globally unique edge that can be added to any graph.
func NewEdge() Edge {
	return &defaultEdge{atomic.AddUint64(&defaultEdgeIndex, 1)}
}

// OppositeVertexOf finds, for any graph, the vertex connected by the given edge
// that is not the given vertex.
func OppositeVertexOf(g Graph, e Edge, v Vertex) (Vertex, error) {
	test, err := g.SourceVertexOf(e)
	if err != nil {
		return nil, err
	}

	if test != v {
		return test, nil
	}

	return g.TargetVertexOf(e)
}

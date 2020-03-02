// Portions of this file are derived from JGraphT, a free Java graph-theory
// library.
//
// (C) Copyright 2003-2016, by Barak Naveh and Contributors.

package graph

import (
	"github.com/puppetlabs/horsehead/v2/datastructure"
)

type intrusiveEdge struct {
	Source, Target Vertex
	Edge           Edge
	Weight         float64
}

type baseGraphOps interface {
	EdgesBetween(source, target Vertex) EdgeSet
	EdgeBetween(source, target Vertex) (Edge, error)
	EdgesOf(vertex Vertex) EdgeSet
	AddEdge(edge Edge)
	RemoveEdge(edge Edge)
	Vertices() MutableVertexSet
}

type baseEdgesView struct {
	g *baseGraph
}

func (sev *baseEdgesView) Contains(edge Edge) bool {
	return sev.g.edges.Contains(edge)
}

func (sev *baseEdgesView) Count() uint {
	return uint(sev.g.edges.Size())
}

func (sev *baseEdgesView) AsSlice() []Edge {
	s := make([]Edge, sev.g.edges.Size())

	i := 0
	sev.ForEach(func(edge Edge) error {
		s[i] = edge
		i++

		return nil
	})

	return s
}

func (sev *baseEdgesView) ForEach(fn EdgeSetIterationFunc) error {
	return sev.g.edges.ForEachInto(func(key Edge, value *intrusiveEdge) error {
		return fn(key)
	})
}

type baseGraph struct {
	AllowsLoops, AllowsMultipleEdges bool
	Ops                              baseGraphOps

	features  GraphFeature
	edges     datastructure.Map // map[Edge]*intrusiveEdge
	edgesView EdgeSet
}

func (g *baseGraph) Features() GraphFeature {
	return g.features
}

func (g *baseGraph) EdgesBetween(source, target Vertex) EdgeSet {
	return g.Ops.EdgesBetween(source, target)
}

func (g *baseGraph) EdgeBetween(source, target Vertex) (Edge, error) {
	return g.Ops.EdgeBetween(source, target)
}

func (g *baseGraph) Connect(source, target Vertex) error {
	return g.AddEdge(source, target, NewEdge())
}

func (g *baseGraph) AddEdge(source, target Vertex, edge Edge) error {
	return g.AddEdgeWithWeight(source, target, edge, DefaultEdgeWeight)
}

func (g *baseGraph) ConnectWithWeight(source, target Vertex, weight float64) error {
	return g.AddEdgeWithWeight(source, target, NewEdge(), weight)
}

func (g *baseGraph) AddEdgeWithWeight(source, target Vertex, edge Edge, weight float64) error {
	if g.ContainsEdge(edge) {
		return ErrEdgeAlreadyInGraph
	}

	if !g.ContainsVertex(source) {
		return &VertexNotFoundError{source}
	}
	if !g.ContainsVertex(target) {
		return &VertexNotFoundError{target}
	}

	if !g.AllowsMultipleEdges && g.ContainsEdgeBetween(source, target) {
		return ErrEdgeAlreadyInGraph
	}

	if !g.AllowsLoops && source == target {
		return ErrWouldCreateLoop
	}

	ie := &intrusiveEdge{
		Source: source,
		Target: target,
		Edge:   edge,
		Weight: weight,
	}

	g.edges.Put(edge, ie)
	g.Ops.AddEdge(edge)

	return nil
}

func (g *baseGraph) AddVertex(vertex Vertex) {
	g.Ops.Vertices().Add(vertex)
}

func (g *baseGraph) ContainsEdgeBetween(source, target Vertex) bool {
	_, err := g.EdgeBetween(source, target)
	return err == nil
}

func (g *baseGraph) ContainsEdge(edge Edge) bool {
	return g.edges.Contains(edge)
}

func (g *baseGraph) ContainsVertex(vertex Vertex) bool {
	return g.Vertices().Contains(vertex)
}

func (g *baseGraph) Edges() EdgeSet {
	if g.edgesView == nil {
		g.edgesView = &baseEdgesView{g}
	}

	return g.edgesView
}

func (g *baseGraph) EdgesOf(vertex Vertex) (EdgeSet, error) {
	if !g.ContainsVertex(vertex) {
		return nil, &VertexNotFoundError{vertex}
	}

	return g.Ops.EdgesOf(vertex), nil
}

func (g *baseGraph) RemoveEdges(edges []Edge) (modified bool) {
	for _, edge := range edges {
		modified = modified || g.RemoveEdge(edge)
	}

	return
}

func (g *baseGraph) RemoveEdgesBetween(source, target Vertex) EdgeSet {
	edges := g.EdgesBetween(source, target)
	g.RemoveEdges(edges.AsSlice())

	return edges
}

func (g *baseGraph) RemoveEdge(edge Edge) bool {
	if !g.ContainsEdge(edge) {
		return false
	}

	g.Ops.RemoveEdge(edge)
	g.edges.Remove(edge)

	return true
}

func (g *baseGraph) RemoveEdgeBetween(source, target Vertex) (Edge, error) {
	edge, err := g.EdgeBetween(source, target)
	if err != nil {
		return nil, err
	}

	g.RemoveEdge(edge)
	return edge, nil
}

func (g *baseGraph) RemoveVertices(vertices []Vertex) (modified bool) {
	for _, vertex := range vertices {
		modified = modified || g.RemoveVertex(vertex)
	}

	return
}

func (g *baseGraph) RemoveVertex(vertex Vertex) bool {
	if !g.ContainsVertex(vertex) {
		return false
	}

	g.RemoveEdges(g.Ops.EdgesOf(vertex).AsSlice())
	g.Ops.Vertices().Remove(vertex)

	return true
}

func (g *baseGraph) Vertices() VertexSet {
	return g.Ops.Vertices()
}

func (g *baseGraph) SourceVertexOf(edge Edge) (Vertex, error) {
	ie, found := g.edges.Get(edge)
	if !found {
		return nil, ErrEdgeNotFound
	}

	return ie.(*intrusiveEdge).Source, nil
}

func (g *baseGraph) TargetVertexOf(edge Edge) (Vertex, error) {
	ie, found := g.edges.Get(edge)
	if !found {
		return nil, ErrEdgeNotFound
	}

	return ie.(*intrusiveEdge).Target, nil
}

func (g *baseGraph) WeightOf(edge Edge) (float64, error) {
	ie, found := g.edges.Get(edge)
	if !found {
		return DefaultEdgeWeight, ErrEdgeNotFound
	}

	return ie.(*intrusiveEdge).Weight, nil
}

func newBaseGraph(features GraphFeature, allowsLoops, allowsMultipleEdges bool, ops baseGraphOps) *baseGraph {
	var edges datastructure.Map
	if features&DeterministicIteration != 0 {
		edges = datastructure.NewLinkedHashMap()
	} else {
		edges = datastructure.NewHashMap()
	}

	return &baseGraph{
		AllowsLoops:         allowsLoops,
		AllowsMultipleEdges: allowsMultipleEdges,
		Ops:                 ops,

		features: features,
		edges:    edges,
	}
}

type baseUndirectedGraph struct {
	*baseGraph
	Ops baseUndirectedGraphOps
}

type baseUndirectedGraphOps interface {
	baseGraphOps
	DegreeOf(vertex Vertex) uint
}

func (ug *baseUndirectedGraph) DegreeOf(vertex Vertex) (uint, error) {
	if !ug.ContainsVertex(vertex) {
		return 0, &VertexNotFoundError{vertex}
	}

	return ug.Ops.DegreeOf(vertex), nil
}

func newBaseUndirectedGraph(features GraphFeature, allowsLoops, allowsMultipleEdges bool, ops baseUndirectedGraphOps) *baseUndirectedGraph {
	return &baseUndirectedGraph{newBaseGraph(features, allowsLoops, allowsMultipleEdges, ops), ops}
}

type baseDirectedGraph struct {
	*baseGraph
	Ops baseDirectedGraphOps
}

type baseDirectedGraphOps interface {
	baseGraphOps
	InDegreeOf(vertex Vertex) uint
	IncomingEdgesOf(vertex Vertex) EdgeSet
	OutDegreeOf(vertex Vertex) uint
	OutgoingEdgesOf(vertex Vertex) EdgeSet
}

func (dg *baseDirectedGraph) InDegreeOf(vertex Vertex) (uint, error) {
	if !dg.ContainsVertex(vertex) {
		return 0, &VertexNotFoundError{vertex}
	}

	return dg.Ops.InDegreeOf(vertex), nil
}

func (dg *baseDirectedGraph) IncomingEdgesOf(vertex Vertex) (EdgeSet, error) {
	if !dg.ContainsVertex(vertex) {
		return nil, &VertexNotFoundError{vertex}
	}

	return dg.Ops.IncomingEdgesOf(vertex), nil
}

func (dg *baseDirectedGraph) OutDegreeOf(vertex Vertex) (uint, error) {
	if !dg.ContainsVertex(vertex) {
		return 0, &VertexNotFoundError{vertex}
	}

	return dg.Ops.OutDegreeOf(vertex), nil
}

func (dg *baseDirectedGraph) OutgoingEdgesOf(vertex Vertex) (EdgeSet, error) {
	if !dg.ContainsVertex(vertex) {
		return nil, &VertexNotFoundError{vertex}
	}

	return dg.Ops.OutgoingEdgesOf(vertex), nil
}

func newBaseDirectedGraph(features GraphFeature, allowsLoops, allowsMultipleEdges bool, ops baseDirectedGraphOps) *baseDirectedGraph {
	return &baseDirectedGraph{newBaseGraph(features, allowsLoops, allowsMultipleEdges, ops), ops}
}

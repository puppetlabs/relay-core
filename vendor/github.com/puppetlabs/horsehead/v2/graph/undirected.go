// Portions of this file are derived from JGraphT, a free Java graph-theory
// library.
//
// (C) Copyright 2015-2016, by Barak Naveh and Contributors.

package graph

import (
	"github.com/puppetlabs/horsehead/v2/datastructure"
)

// UndirectedGraphSupportedFeatures are the features supported by all undirected
// graphs.
const UndirectedGraphSupportedFeatures = DeterministicIteration

type undirectedVertexSet struct {
	features GraphFeature
	storage  datastructure.Map // map[Vertex]MutableEdgeSet
}

func (vs *undirectedVertexSet) Contains(vertex Vertex) bool {
	return vs.storage.Contains(vertex)
}

func (vs *undirectedVertexSet) Count() uint {
	return uint(vs.storage.Size())
}

func (vs *undirectedVertexSet) AsSlice() []Vertex {
	s := make([]Vertex, 0, vs.Count())
	vs.storage.KeysInto(&s)
	return s
}

func (vs *undirectedVertexSet) ForEach(fn VertexSetIterationFunc) error {
	return vs.storage.ForEachInto(func(key Vertex, value MutableEdgeSet) error {
		return fn(key)
	})
}

func (vs *undirectedVertexSet) Add(vertex Vertex) {
	if vs.storage.Contains(vertex) {
		return
	}

	vs.storage.Put(vertex, nil)
}

func (vs *undirectedVertexSet) Remove(vertex Vertex) {
	vs.storage.Remove(vertex)
}

func (vs *undirectedVertexSet) edgesOf(vertex Vertex) MutableEdgeSet {
	if !vs.storage.Contains(vertex) {
		return nil
	}

	var set MutableEdgeSet
	vs.storage.GetInto(vertex, &set)

	if set == nil {
		if vs.features&DeterministicIteration != 0 {
			set = NewMutableEdgeSet(datastructure.NewLinkedHashSet())
		} else {
			set = NewMutableEdgeSet(datastructure.NewHashSet())
		}

		vs.storage.Put(vertex, set)
	}

	return set
}

type undirectedGraphOps struct {
	g        *baseUndirectedGraph
	vertices *undirectedVertexSet
}

func (o *undirectedGraphOps) EdgesBetween(source, target Vertex) EdgeSet {
	if !o.g.ContainsVertex(source) || !o.g.ContainsVertex(target) {
		return nil
	}

	es := &unenforcedSliceEdgeSet{}

	o.vertices.edgesOf(source).ForEach(func(edge Edge) error {
		if o.edgeHasSourceAndTarget(edge, source, target) {
			es.Add(edge)
		}

		return nil
	})

	return es
}

func (o *undirectedGraphOps) EdgeBetween(source, target Vertex) (Edge, error) {
	if !o.g.ContainsVertex(source) || !o.g.ContainsVertex(target) {
		return nil, &NotConnectedError{Source: source, Target: target}
	}

	var found Edge
	err := o.vertices.edgesOf(source).ForEach(func(edge Edge) error {
		if o.edgeHasSourceAndTarget(edge, source, target) {
			found = edge
			return datastructure.ErrStopIteration
		}

		return nil
	})

	if err == datastructure.ErrStopIteration {
		return found, nil
	}

	return nil, &NotConnectedError{Source: source, Target: target}
}

func (o *undirectedGraphOps) edgeHasSourceAndTarget(edge Edge, source, target Vertex) bool {
	ts, _ := o.g.SourceVertexOf(edge)
	tt, _ := o.g.TargetVertexOf(edge)

	return (source == ts && target == tt) || (source == tt && target == ts)
}

func (o *undirectedGraphOps) EdgesOf(vertex Vertex) EdgeSet {
	if !o.g.ContainsVertex(vertex) {
		return nil
	}

	return o.vertices.edgesOf(vertex)
}

func (o *undirectedGraphOps) AddEdge(edge Edge) {
	source, _ := o.g.SourceVertexOf(edge)
	target, _ := o.g.TargetVertexOf(edge)

	o.vertices.edgesOf(source).Add(edge)
	o.vertices.edgesOf(target).Add(edge)
}

func (o *undirectedGraphOps) RemoveEdge(edge Edge) {
	source, _ := o.g.SourceVertexOf(edge)
	target, _ := o.g.TargetVertexOf(edge)

	o.vertices.edgesOf(source).Remove(edge)
	o.vertices.edgesOf(target).Remove(edge)
}

func (o *undirectedGraphOps) Vertices() MutableVertexSet {
	return o.vertices
}

func (o *undirectedGraphOps) DegreeOf(vertex Vertex) uint {
	if !o.g.ContainsVertex(vertex) {
		return 0
	}

	return o.vertices.edgesOf(vertex).Count()
}

func newUndirectedGraph(features GraphFeature, allowLoops, allowMultipleEdges bool) *baseUndirectedGraph {
	var vertexStorage datastructure.Map
	if features&DeterministicIteration != 0 {
		vertexStorage = datastructure.NewLinkedHashMap()
	} else {
		vertexStorage = datastructure.NewHashMap()
	}

	ops := &undirectedGraphOps{
		vertices: &undirectedVertexSet{features: features, storage: vertexStorage},
	}

	g := newBaseUndirectedGraph(features, allowLoops, allowMultipleEdges, ops)
	ops.g = g

	return g
}

//
// Simple graphs
//

// A SimpleGraph is an undirected graph that does not permit loops or multiple
// edges between vertices.
type SimpleGraph struct {
	MutableUndirectedGraph
}

// NewSimpleGraph creates a new simple graph.
func NewSimpleGraph() *SimpleGraph {
	return NewSimpleGraphWithFeatures(0)
}

// NewSimpleGraphWithFeatures creates a new simple graph with the specified
// graph features.
func NewSimpleGraphWithFeatures(features GraphFeature) *SimpleGraph {
	return &SimpleGraph{newUndirectedGraph(features&UndirectedGraphSupportedFeatures, false, false)}
}

// A SimpleWeightedGraph is a simple graph for which edges are assigned weights.
type SimpleWeightedGraph struct {
	MutableUndirectedWeightedGraph
}

// NewSimpleWeightedGraph creates a new simple weighted graph.
func NewSimpleWeightedGraph() *SimpleWeightedGraph {
	return NewSimpleWeightedGraphWithFeatures(0)
}

// NewSimpleWeightedGraphWithFeatures creates a new simple weighted graph with
// the specified graph features.
func NewSimpleWeightedGraphWithFeatures(features GraphFeature) *SimpleWeightedGraph {
	return &SimpleWeightedGraph{newUndirectedGraph(features&UndirectedGraphSupportedFeatures, false, false)}
}

//
// Multigraphs
//

// An UndirectedMultigraph is an undirected graph that does not permit loops,
// but does permit multiple edges between any two vertices.
type UndirectedMultigraph struct {
	MutableUndirectedGraph
}

// NewUndirectedMultigraph creates a new undirected multigraph.
func NewUndirectedMultigraph() *UndirectedMultigraph {
	return NewUndirectedMultigraphWithFeatures(0)
}

// NewUndirectedMultigraphWithFeatures creates a new undirected multigraph with
// the specified graph features.
func NewUndirectedMultigraphWithFeatures(features GraphFeature) *UndirectedMultigraph {
	return &UndirectedMultigraph{newUndirectedGraph(features&UndirectedGraphSupportedFeatures, false, true)}
}

// An UndirectedWeightedMultigraph is an undirected multigraph for which edges
// are assigned weights.
type UndirectedWeightedMultigraph struct {
	MutableUndirectedWeightedGraph
}

// NewUndirectedWeightedMultigraph creates a new undirected weighted multigraph.
func NewUndirectedWeightedMultigraph() *UndirectedWeightedMultigraph {
	return NewUndirectedWeightedMultigraphWithFeatures(0)
}

// NewUndirectedWeightedMultigraphWithFeatures creates a new undirected weighted
// multigraph with the specified graph features.
func NewUndirectedWeightedMultigraphWithFeatures(features GraphFeature) *UndirectedWeightedMultigraph {
	return &UndirectedWeightedMultigraph{newUndirectedGraph(features&UndirectedGraphSupportedFeatures, false, true)}
}

//
// Pseudographs
//

// An UndirectedPseudograph is an undirected graph that permits both loops and
// multiple edges between vertices.
type UndirectedPseudograph struct {
	MutableUndirectedGraph
}

// NewUndirectedPseudograph creates a new undirected pseudograph.
func NewUndirectedPseudograph() *UndirectedPseudograph {
	return NewUndirectedPseudographWithFeatures(0)
}

// NewUndirectedPseudographWithFeatures creates a new undirected pseudograph
// with the specified graph features.
func NewUndirectedPseudographWithFeatures(features GraphFeature) *UndirectedPseudograph {
	return &UndirectedPseudograph{newUndirectedGraph(features&UndirectedGraphSupportedFeatures, true, true)}
}

// An UndirectedWeightedPseudograph is an undirected pseudograph for which edges
// are assigned weights.
type UndirectedWeightedPseudograph struct {
	MutableUndirectedWeightedGraph
}

// NewUndirectedWeightedPseudograph creates a new undirected weighted
// pseudograph.
func NewUndirectedWeightedPseudograph() *UndirectedWeightedPseudograph {
	return NewUndirectedWeightedPseudographWithFeatures(0)
}

// NewUndirectedWeightedPseudographWithFeatures creates a new undirected
// weighted pseudograph with the specified graph features.
func NewUndirectedWeightedPseudographWithFeatures(features GraphFeature) *UndirectedWeightedPseudograph {
	return &UndirectedWeightedPseudograph{newUndirectedGraph(features&UndirectedGraphSupportedFeatures, true, true)}
}

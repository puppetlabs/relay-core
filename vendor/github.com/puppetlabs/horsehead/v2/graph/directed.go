// Portions of this file are derived from JGraphT, a free Java graph-theory
// library.
//
// (C) Copyright 2015-2017, by Barak Naveh and Contributors.

package graph

import (
	"github.com/puppetlabs/horsehead/v2/datastructure"
)

// DirectedGraphSupportedFeatures are the features supported by all directed
// graphs.
const DirectedGraphSupportedFeatures = DeterministicIteration

type directedEdgeContainer struct {
	incoming MutableEdgeSet
	outgoing MutableEdgeSet
}

type directedVertexSet struct {
	features GraphFeature
	storage  datastructure.Map // map[Vertex]*directedEdgeContainer
}

func (vs *directedVertexSet) Contains(vertex Vertex) bool {
	return vs.storage.Contains(vertex)
}

func (vs *directedVertexSet) Count() uint {
	return uint(vs.storage.Size())
}

func (vs *directedVertexSet) AsSlice() []Vertex {
	s := make([]Vertex, 0, vs.Count())
	vs.storage.KeysInto(&s)
	return s
}

func (vs *directedVertexSet) ForEach(fn VertexSetIterationFunc) error {
	return vs.storage.ForEachInto(func(key Vertex, value *directedEdgeContainer) error {
		return fn(key)
	})
}

func (vs *directedVertexSet) Add(vertex Vertex) {
	if vs.storage.Contains(vertex) {
		return
	}

	vs.storage.Put(vertex, nil)
}

func (vs *directedVertexSet) Remove(vertex Vertex) {
	vs.storage.Remove(vertex)
}

func (vs *directedVertexSet) initVertex(vertex Vertex) *directedEdgeContainer {
	if !vs.storage.Contains(vertex) {
		return nil
	}

	var container *directedEdgeContainer
	vs.storage.GetInto(vertex, &container)

	if container == nil {
		container = &directedEdgeContainer{}
		vs.storage.Put(vertex, container)
	}

	return container
}

func (vs *directedVertexSet) incomingEdgesOf(vertex Vertex) MutableEdgeSet {
	container := vs.initVertex(vertex)
	if container == nil {
		return nil
	}

	if container.incoming == nil {
		if vs.features&DeterministicIteration != 0 {
			container.incoming = NewMutableEdgeSet(datastructure.NewLinkedHashSet())
		} else {
			container.incoming = NewMutableEdgeSet(datastructure.NewHashSet())
		}
	}

	return container.incoming
}

func (vs *directedVertexSet) outgoingEdgesOf(vertex Vertex) MutableEdgeSet {
	container := vs.initVertex(vertex)
	if container == nil {
		return nil
	}

	if container.outgoing == nil {
		if vs.features&DeterministicIteration != 0 {
			container.outgoing = NewMutableEdgeSet(datastructure.NewLinkedHashSet())
		} else {
			container.outgoing = NewMutableEdgeSet(datastructure.NewHashSet())
		}
	}

	return container.outgoing
}

type directedGraphOps struct {
	g        *baseDirectedGraph
	vertices *directedVertexSet
}

func (o *directedGraphOps) EdgesBetween(source, target Vertex) EdgeSet {
	if !o.g.ContainsVertex(source) || !o.g.ContainsVertex(target) {
		return nil
	}

	es := &unenforcedSliceEdgeSet{}

	o.vertices.outgoingEdgesOf(source).ForEach(func(edge Edge) error {
		tt, _ := o.g.TargetVertexOf(edge)
		if tt == target {
			es.Add(edge)
		}

		return nil
	})

	return es
}

func (o *directedGraphOps) EdgeBetween(source, target Vertex) (Edge, error) {
	if !o.g.ContainsVertex(source) || !o.g.ContainsVertex(target) {
		return nil, &NotConnectedError{Source: source, Target: target}
	}

	var found Edge
	err := o.vertices.outgoingEdgesOf(source).ForEach(func(edge Edge) error {
		tt, _ := o.g.TargetVertexOf(edge)
		if tt == target {
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

func (o *directedGraphOps) EdgesOf(vertex Vertex) EdgeSet {
	if !o.g.ContainsVertex(vertex) {
		return nil
	}

	var set MutableEdgeSet
	if o.g.Features()&DeterministicIteration != 0 {
		set = NewMutableEdgeSet(datastructure.NewLinkedHashSet())
	} else {
		set = NewMutableEdgeSet(datastructure.NewHashSet())
	}

	o.IncomingEdgesOf(vertex).ForEach(func(edge Edge) error {
		set.Add(edge)
		return nil
	})
	o.OutgoingEdgesOf(vertex).ForEach(func(edge Edge) error {
		set.Add(edge)
		return nil
	})

	return set
}

func (o *directedGraphOps) AddEdge(edge Edge) {
	source, _ := o.g.SourceVertexOf(edge)
	target, _ := o.g.TargetVertexOf(edge)

	o.vertices.outgoingEdgesOf(source).Add(edge)
	o.vertices.incomingEdgesOf(target).Add(edge)
}

func (o *directedGraphOps) RemoveEdge(edge Edge) {
	source, _ := o.g.SourceVertexOf(edge)
	target, _ := o.g.TargetVertexOf(edge)

	o.vertices.outgoingEdgesOf(source).Remove(edge)
	o.vertices.incomingEdgesOf(target).Remove(edge)
}

func (o *directedGraphOps) Vertices() MutableVertexSet {
	return o.vertices
}

func (o *directedGraphOps) InDegreeOf(vertex Vertex) uint {
	if !o.g.ContainsVertex(vertex) {
		return 0
	}

	return o.vertices.incomingEdgesOf(vertex).Count()
}

func (o *directedGraphOps) IncomingEdgesOf(vertex Vertex) EdgeSet {
	if !o.g.ContainsVertex(vertex) {
		return nil
	}

	return o.vertices.incomingEdgesOf(vertex)
}

func (o *directedGraphOps) OutDegreeOf(vertex Vertex) uint {
	if !o.g.ContainsVertex(vertex) {
		return 0
	}

	return o.vertices.outgoingEdgesOf(vertex).Count()
}

func (o *directedGraphOps) OutgoingEdgesOf(vertex Vertex) EdgeSet {
	if !o.g.ContainsVertex(vertex) {
		return nil
	}

	return o.vertices.outgoingEdgesOf(vertex)
}

func newDirectedGraph(features GraphFeature, allowLoops, allowMultipleEdges bool) *baseDirectedGraph {
	var vertexStorage datastructure.Map
	if features&DeterministicIteration != 0 {
		vertexStorage = datastructure.NewLinkedHashMap()
	} else {
		vertexStorage = datastructure.NewHashMap()
	}

	ops := &directedGraphOps{
		vertices: &directedVertexSet{features: features, storage: vertexStorage},
	}

	g := newBaseDirectedGraph(features, allowLoops, allowMultipleEdges, ops)
	ops.g = g

	return g
}

//
// Simple graphs
//

// A SimpleDirectedGraph is a directed graph that does not permit loops or
// multiple edges between vertices.
type SimpleDirectedGraph struct {
	MutableDirectedGraph
}

// NewSimpleDirectedGraph creates a new simple directed graph.
func NewSimpleDirectedGraph() *SimpleDirectedGraph {
	return NewSimpleDirectedGraphWithFeatures(0)
}

// NewSimpleDirectedGraphWithFeatures creates a new simple directed graph with
// the specified graph features.
func NewSimpleDirectedGraphWithFeatures(features GraphFeature) *SimpleDirectedGraph {
	return &SimpleDirectedGraph{newDirectedGraph(features&DirectedGraphSupportedFeatures, false, false)}
}

// A SimpleDirectedWeightedGraph is a simple directed graph for which edges are
// assigned weights.
type SimpleDirectedWeightedGraph struct {
	MutableDirectedWeightedGraph
}

// NewSimpleDirectedWeightedGraph creates a new simple directed weighted graph.
func NewSimpleDirectedWeightedGraph() *SimpleDirectedWeightedGraph {
	return NewSimpleDirectedWeightedGraphWithFeatures(0)
}

// NewSimpleDirectedWeightedGraphWithFeatures creates a new simple directed
// weighted graph with the specified graph features.
func NewSimpleDirectedWeightedGraphWithFeatures(features GraphFeature) *SimpleDirectedWeightedGraph {
	return &SimpleDirectedWeightedGraph{newDirectedGraph(features&DirectedGraphSupportedFeatures, false, false)}
}

//
// Multigraphs
//

// A DirectedMultigraph is a directed graph that does not permit loops, but does
// permit multiple edges between any two vertices.
type DirectedMultigraph struct {
	MutableDirectedGraph
}

// NewDirectedMultigraph creates a new directed multigraph.
func NewDirectedMultigraph() *DirectedMultigraph {
	return NewDirectedMultigraphWithFeatures(0)
}

// NewDirectedMultigraphWithFeatures creates a new directed multigraph with the
// specified graph features.
func NewDirectedMultigraphWithFeatures(features GraphFeature) *DirectedMultigraph {
	return &DirectedMultigraph{newDirectedGraph(features&DirectedGraphSupportedFeatures, false, true)}
}

// A DirectedWeightedMultigraph is a directed multigraph for which edges are
// assigned weights.
type DirectedWeightedMultigraph struct {
	MutableDirectedWeightedGraph
}

// NewDirectedWeightedMultigraph creates a new directed weighted multigraph.
func NewDirectedWeightedMultigraph() *DirectedWeightedMultigraph {
	return NewDirectedWeightedMultigraphWithFeatures(0)
}

// NewDirectedWeightedMultigraphWithFeatures creates a new directed weighted
// multigraph with the specified graph features.
func NewDirectedWeightedMultigraphWithFeatures(features GraphFeature) *DirectedWeightedMultigraph {
	return &DirectedWeightedMultigraph{newDirectedGraph(features&DirectedGraphSupportedFeatures, false, true)}
}

//
// Pseudographs
//

// A DirectedPseudograph is a directed graph that permits both loops and
// multiple edges between vertices.
type DirectedPseudograph struct {
	MutableDirectedGraph
}

// NewDirectedPseudograph creates a new directed pseudograph.
func NewDirectedPseudograph() *DirectedPseudograph {
	return NewDirectedPseudographWithFeatures(0)
}

// NewDirectedPseudographWithFeatures a new directed pseudograph with the given
// graph features.
func NewDirectedPseudographWithFeatures(features GraphFeature) *DirectedPseudograph {
	return &DirectedPseudograph{newDirectedGraph(features&DirectedGraphSupportedFeatures, true, true)}
}

// A DirectedWeightedPseudograph is a directed pseudograph for which the edges
// are assigned weights.
type DirectedWeightedPseudograph struct {
	MutableDirectedWeightedGraph
}

// NewDirectedWeightedPseudograph creates a new directed weighted pseudograph.
func NewDirectedWeightedPseudograph() *DirectedWeightedPseudograph {
	return NewDirectedWeightedPseudographWithFeatures(0)
}

// NewDirectedWeightedPseudographWithFeatures creates a new directed weighted
// pseudograph with the given graph features.
func NewDirectedWeightedPseudographWithFeatures(features GraphFeature) *DirectedWeightedPseudograph {
	return &DirectedWeightedPseudograph{newDirectedGraph(features&DirectedGraphSupportedFeatures, true, true)}
}

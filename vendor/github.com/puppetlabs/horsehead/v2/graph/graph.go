// Portions of this file are derived from JGraphT, a free Java graph-theory
// library.
//
// (C) Copyright 2003-2016, by Barak Naveh and Contributors.

// Package gographt provides interfaces and data structures that describe
// discrete graphs. It is inspired by the excellent JGraphT library
// (http://jgrapht.org/) and makes use of some applicable parts of it. Note that
// there are licensing constraints on JGraphT that may apply to this package as
// well; you may assume some risk by using it. See
// https://github.com/jgrapht/jgrapht/wiki/Relicensing for more information.
//
// This package supports simple graphs, multigraphs, and pseudographs in both
// directed and undirected variants. Each of these variants may be weighted or
// unweighted.
//
// A unique feature of this package is support for deterministic iteration; that
// is, each graph can retain information about the order of vertices and edges
// added to it, and given an immutable copy, iterate in a consistent order (not
// necessarily that of insertion, though). Most algorithms implemented also
// support deterministic iteration.
//
// In general, public graph interfaces are immutable and the types returned by
// constructor methods are mutable. Algorithms that accept graphs will always
// use the immutable interfaces and clone the graph if needed to perform
// computations.
package graph

// A Vertex is the type of a node used in a graph.
//
// Vertices may be any type; they are distinguished from interface{} only for
// clarity. When adding a vertex to a graph, the Go language specification
// comparison rules apply to determine whether the vertex already exists.
type Vertex interface{}

// VertexSetIterationFunc is a callback function used by the ForEach method of a
// vertex set.
//
// This function can return datastructure.StopIteration to break from the iteration.
type VertexSetIterationFunc func(vertex Vertex) error

// A VertexSet is a read-only collection of vertices.
type VertexSet interface {
	// Contains returns true if the given vertex exists in this set, and false
	// otherwise.
	Contains(vertex Vertex) bool

	// Count returns the number of vertices in this set.
	Count() uint

	// AsSlice returns all the vertices in this set as a slice.
	AsSlice() []Vertex

	// ForEach iterates over each vertex in this set, invoking the given
	// function on each iteration. If the function returns an error, iteration
	// is stopped and the error is returned.
	ForEach(fn VertexSetIterationFunc) error
}

// A MutableVertexSet is an extension of VertexSet that additionally supports
// adding and removing vertices.
type MutableVertexSet interface {
	VertexSet

	// Add adds the given vertex to this set. If the vertex already exists in
	// this set, it is not duplicated.
	Add(vertex Vertex)

	// Remove removes the given vertex from this set if it exists.
	Remove(vertex Vertex)
}

// An Edge is the type of a connection between vertices.
//
// Edges may be any type; they are distinguished from interface{} only for
// clarity. When adding an edge to a graph, the Go language specification rules
// apply to determine whether the edge already exists.
type Edge interface{}

// EdgeSetIterationFunc is a callback function used by the ForEach method of an
// edge set.
//
// This function can return datastructure.StopIteration to break from the iteration.
type EdgeSetIterationFunc func(edge Edge) error

// An EdgeSet is a read-only collection of edges.
type EdgeSet interface {
	// Contains returns true if the given edge exists in this set, and false
	// otherwise.
	Contains(edge Edge) bool

	// Count returns the number of edges in this set.
	Count() uint

	// AsSlice returns all the edges in this set as a slice.
	AsSlice() []Edge

	// ForEach iterates over each edge in this set, invoking the given function
	// on each iteration. If the function returns an error, iteration is stopped
	// and the error is returned.
	ForEach(fn EdgeSetIterationFunc) error
}

// A MutableEdgeSet is an extension of EdgeSet that additionally supports adding
// and removing edges.
type MutableEdgeSet interface {
	EdgeSet

	// Add adds the given edge to this set. If the edge already exists in this
	// set, it is not duplicated.
	Add(edge Edge)

	// Remove removes the given edge from this set if it exists.
	Remove(edge Edge)
}

// DefaultEdgeWeight is the weight of an edge connected without explicitly
// specifying a weight in a weighted graph.
const DefaultEdgeWeight = float64(1.)

// A Graph is a structure that contains a collection of vertices, some of which
// may be related to each other by edges.
type Graph interface {
	// Features returns the graph features being used for this graph.
	Features() GraphFeature

	// EdgesBetween finds all edges that connect the given vertices and returns
	// a read-only view of them. If the DeterministicIteration feature is used
	// in this graph, the edge set will always iterate over the edges in the
	// same order.
	EdgesBetween(source, target Vertex) EdgeSet

	// EdgeBetween finds an arbitrary edge that connects the given vertices and
	// returns it. If the DeterministicIteration feature is used in this graph,
	// the returned edge will always be the same. If no edge connects the given
	// vertices, an error of type NotConnectedError is returned. If prior
	// knowledge ensures the vertices are connected by at least one edge, the
	// error from this method can be safely ignored.
	EdgeBetween(source, target Vertex) (Edge, error)

	// ContainsEdgeBetween returns true if at least one edge connects the given
	// source and target vertices, and false otherwise.
	ContainsEdgeBetween(source, target Vertex) bool

	// ContainsEdge returns true if the given edge exists in this graph, and
	// false otherwise.
	ContainsEdge(edge Edge) bool

	// ContainsVertex returns true if the given vertex exists in this graph, and
	// false otherwise.
	ContainsVertex(vertex Vertex) bool

	// Edges returns a read-only view of all edges in this graph. If the
	// DeterministicIteration feature is used in this graph, the edge set will
	// always iterate over the edges in the same order.
	Edges() EdgeSet

	// EdgesOf returns a read-only view of all edges connected to the given
	// vertex regardless of their direction. If the given vertex does not exist
	// in this graph, an error of type VertexNotFoundError is returned. If prior
	// knowledge ensures the vertex exists in the graph, the error from this
	// method can be safely ignored. If the DeterministicIteration feature is
	// used in this graph, the edge set will always iterate over the edges in
	// the same order.
	EdgesOf(vertex Vertex) (EdgeSet, error)

	// Vertices returns a read-only view of all vertices in this graph. If the
	// DeterministicIteration feature is used in this graph, the vertex set will
	// always iterate over the vertices in the same order.
	Vertices() VertexSet

	// SourceVertexOf returns the source vertex for a given edge. If the edge
	// does not exist in the graph, ErrEdgeNotFound is returned. If prior
	// knowledge ensures the edge exists in the graph, the error from this
	// method can be safely ignored.
	SourceVertexOf(edge Edge) (Vertex, error)

	// TargetVertexOf returns the target vertex for a given edge. If the edge
	// does not exist in the graph, ErrEdgeNotFound is returned. If prior
	// knowledge ensures the edge exists in the graph, the error from this
	// method can be safely ignored.
	TargetVertexOf(edge Edge) (Vertex, error)

	// WeightOf returns the weight associated with a given edge. For unweighted
	// graphs, this is always the same as DefaultEdgeWeight. If the edge does
	// not exist in the graph, ErrEdgeNotFound is returned. If prior knowledge
	// ensures the edge exists in the graph, the error from this method can be
	// safely ignored.
	WeightOf(edge Edge) (float64, error)
}

// Mutable is a graph mixin that allows graphs to be modified.
type Mutable interface {
	// Connect adds an edge between the given source and target vertices. The
	// edge is created using the NewEdge function. If this graph does not
	// contain either the source or target vertices, an error of type
	// VertexNotFoundError is returned. If this graph does not permit multiple
	// edges and the source and target vertices are already connected,
	// ErrEdgeAlreadyInGraph is returned. If this graph does not permit loops
	// and the source and target vertices are the same, ErrWouldCreateLoop is
	// returned.
	Connect(source, target Vertex) error

	// AddEdge adds the given edge between the given source and target vertices.
	// If this graph already contains the given edge, ErrEdgeAlreadyInGraph is
	// returned. If this graph does not contain either the source or target
	// vertices, an error of type VertexNotFoundError is returned. If this graph
	// does not permit multiple edges and the source and target vertices are
	// already connected, ErrEdgeAlreadyInGraph is returned. If this graph does
	// not permit loops and the source and target vertices are the same,
	// ErrWouldCreateLoop is returned.
	AddEdge(source, target Vertex, edge Edge) error

	// AddVertex adds the given vertex to the graph. If the vertex already
	// exists in the graph, it is not duplicated.
	AddVertex(vertex Vertex)

	// RemoveEdges removes the given edges from the graph if they exists. It
	// returns true if the graph was modified, and false otherwise.
	RemoveEdges(edges []Edge) bool

	// RemoveEdgesBetween removes all edges that connect the given source and
	// target vertices and returns a read-only view of the removed edges. If the
	// DeterministicIteration feature is used by this graph, the edge set will
	// always iterate over the edges in the same order.
	RemoveEdgesBetween(source, target Vertex) EdgeSet

	// RemoveEdge removes the given edge from the graph if it exists. It returns
	// true if the graph was modified, and false otherwise.
	RemoveEdge(edge Edge) bool

	// RemoveEdgeBetween removes an arbitrary edge connecting the given source
	// and target vertices from the graph. If no edge connects the given
	// vertices, an error of type NotConnectedError is returned. Otherwise, the
	// removed edge is returned. If the DeterministicIteration feature is used
	// by this graph, repeated calls to this function will remove edges in the
	// same order.
	RemoveEdgeBetween(source, target Vertex) (Edge, error)

	// RemoveVertices removes the given vertices from this graph. It returns
	// true if the graph was modified, and false otherwise.
	RemoveVertices(vertices []Vertex) bool

	// RemoveVertex removes the given vertex from this graph. It returns true if
	// the graph was modified, and false otherwise.
	RemoveVertex(vertex Vertex) bool
}

// A DirectedGraph is a graph for which the direction of an edge's assocation to
// its vertices is important.
type DirectedGraph interface {
	Graph

	// InDegreeOf returns the number of edges directed toward the given vertex.
	// If the vertex does not exist in the graph, an error of type
	// VertexNotFoundError is returned. If prior knowledge ensures the vertex
	// exists in the graph, the error from this method can be safely ignored.
	InDegreeOf(vertex Vertex) (uint, error)

	// IncomingEdgesOf returns a read-only view of the edges that are directed
	// toward the given vertex. If the vertex does not exist in the graph, an
	// error of type VertexNotFoundError is returned. If prior knowledge ensures
	// the vertex exists in the graph, the error from this method can be safely
	// ignored.
	IncomingEdgesOf(vertex Vertex) (EdgeSet, error)

	// OutDegreeOf returns the number of edges directed outward from the given
	// vertex. If the vertex does not exist in the graph, an error of type
	// VertexNotFoundError is returned. If prior knowledge ensures the vertex
	// exists in the graph, the error from this method can be safely ignored.
	OutDegreeOf(vertex Vertex) (uint, error)

	// OutgoingEdgesOf returns a read-only view of the edges that are directed
	// outward from the given vertex. If the vertex does not exist in the graph,
	// an error of type VertexNotFoundError is returned. If prior knowledge
	// ensures the vertex exists in the graph, the error from this method can be
	// safely ignored.
	OutgoingEdgesOf(vertex Vertex) (EdgeSet, error)
}

// A MutableDirectedGraph is a directed graph that supports modification.
type MutableDirectedGraph interface {
	DirectedGraph
	Mutable
}

// An UndirectedGraph is a graph for which the direction of an edge's assocation
// to its vertices is not important.
type UndirectedGraph interface {
	Graph

	// DegreeOf returns the number of edges connected to the given vertex. If
	// the vertex does not exist in the graph, an error of type
	// VertexNotFoundError is returned. If prior knowledge ensures the vertex
	// exists in the graph, the error from this method can be safely ignored.
	DegreeOf(vertex Vertex) (uint, error)
}

// A MutableUndirectedGraph is an undirected graph that supports modification.
type MutableUndirectedGraph interface {
	UndirectedGraph
	Mutable
}

// Weighted is a graph mixin that allows edges to be assigned arbitrary
// numerical weights. These weights can be important in evaluating certain
// algorithms.
type Weighted interface {
	// ConnectWithWeight associates the given source and target vertices with
	// each other using the same mechanism as Connect. The given weight is used
	// to label the resulting connection.
	ConnectWithWeight(source, target Vertex, weight float64) error

	// AddEdgeWithWeight associates the given source and target vertices with
	// each other using the given edge. The given weight is used to label the
	// resulting connection.
	AddEdgeWithWeight(source, target Vertex, edge Edge, weight float64) error
}

// A MutableDirectedWeightedGraph is a directed weighted graph that supports
// modification.
type MutableDirectedWeightedGraph interface {
	DirectedGraph
	Weighted
	Mutable
}

// A MutableUndirectedWeightedGraph is an undirected weighted graph that
// supports modification.
type MutableUndirectedWeightedGraph interface {
	UndirectedGraph
	Weighted
	Mutable
}

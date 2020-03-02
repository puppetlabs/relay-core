package graph

// A GraphFeature is a mathematically transparent option for a graph. Graph
// features enable desireable application-specific functionality for a given
// graph (at the possible expense of performance).
type GraphFeature uint

const (
	// DeterministicIteration is a graph feature that causes the insertion order
	// of vertices and edges for a graph to be retained. As a result, iteration
	// order over the vertices and edges will be the same, and many algorithms
	// will be evaluated identically for identical graph constructions.
	//
	// This feature is particularly useful for debugging or for unit tests.
	DeterministicIteration GraphFeature = 1 << iota
)

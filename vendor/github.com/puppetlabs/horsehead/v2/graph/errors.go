package graph

import (
	"errors"
	"fmt"
)

// VertexNotFoundError indicates that an operation involving a given vertex
// could not be completed because that vertex does not exist in the graph.
type VertexNotFoundError struct {
	Vertex Vertex
}

func (e *VertexNotFoundError) Error() string {
	return fmt.Sprintf("graph: vertex %q does not exist", e.Vertex)
}

// NotConnectedError indicates that an operation involving two vertices could
// not be completed because no edges connect those vertices.
type NotConnectedError struct {
	Source, Target Vertex
}

func (e *NotConnectedError) Error() string {
	return fmt.Sprintf("graph: not connected: %q and %q", e.Source, e.Target)
}

var (
	// ErrEdgeAlreadyInGraph indicates that an edge could not be added to a
	// graph because an edge already exists in the graph.
	ErrEdgeAlreadyInGraph = errors.New("graph: edge already present")

	// ErrEdgeNotFound indicates that an operation involving an edge could not
	// be completed because that edge does not exist in the graph.
	ErrEdgeNotFound = errors.New("graph: edge does not exist")

	// ErrWouldCreateLoop indicates that the addition of an edge would create a
	// loop in the graph, and the graph does not support loops.
	ErrWouldCreateLoop = errors.New("graph: loop would be created by edge")
)

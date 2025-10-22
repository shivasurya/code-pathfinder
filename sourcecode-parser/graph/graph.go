package graph

// NewCodeGraph creates and initializes a new CodeGraph instance.
func NewCodeGraph() *CodeGraph {
	return &CodeGraph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}
}

// AddNode adds a node to the code graph.
func (g *CodeGraph) AddNode(node *Node) {
	g.Nodes[node.ID] = node
}

// AddEdge adds an edge between two nodes in the code graph.
func (g *CodeGraph) AddEdge(from, to *Node) {
	edge := &Edge{From: from, To: to}
	g.Edges = append(g.Edges, edge)
	from.OutgoingEdges = append(from.OutgoingEdges, edge)
}

// FindNodesByType finds all nodes of a given type.
func (g *CodeGraph) FindNodesByType(nodeType string) []*Node {
	var nodes []*Node
	for _, node := range g.Nodes {
		if node.Type == nodeType {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

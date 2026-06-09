package query

import "github.com/ogatalars/codegraph-go/internal/store"

type EdgeResult struct {
	FromFQN string
	ToFQN   string
	Kind    string
	File    string
	Line    int
}

func (e *Engine) Callers(fqn string, _ int) ([]EdgeResult, error) {
	rows, err := e.store.GetCallers(fqn)
	if err != nil {
		return nil, err
	}
	return toEdgeResults(rows), nil
}

func (e *Engine) Callees(fqn string, _ int) ([]EdgeResult, error) {
	rows, err := e.store.GetCallees(fqn)
	if err != nil {
		return nil, err
	}
	return toEdgeResults(rows), nil
}

// Trace finds a call path from fromFQN to toFQN via BFS over edges (max depth 10).
func (e *Engine) Trace(fromFQN, toFQN string) ([]string, error) {
	visited := map[string]string{fromFQN: ""}
	queue := []string{fromFQN}

	for depth := 0; depth < 10 && len(queue) > 0; depth++ {
		var next []string
		for _, node := range queue {
			edges, err := e.store.GetCallees(node)
			if err != nil {
				return nil, err
			}
			for _, edge := range edges {
				if _, seen := visited[edge.ToFQN]; seen {
					continue
				}
				visited[edge.ToFQN] = node
				if edge.ToFQN == toFQN {
					return reconstructPath(visited, fromFQN, toFQN), nil
				}
				next = append(next, edge.ToFQN)
			}
		}
		queue = next
	}
	return nil, nil
}

func reconstructPath(parent map[string]string, from, to string) []string {
	var path []string
	for curr := to; curr != ""; curr = parent[curr] {
		path = append([]string{curr}, path...)
	}
	return path
}

func toEdgeResults(rows []store.EdgeRow) []EdgeResult {
	result := make([]EdgeResult, len(rows))
	for i, r := range rows {
		result[i] = EdgeResult{
			FromFQN: r.FromFQN,
			ToFQN:   r.ToFQN,
			Kind:    r.Kind,
			Line:    r.Line,
			File:    r.FilePath,
		}
	}
	return result
}

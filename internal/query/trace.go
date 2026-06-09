package query

type EdgeResult struct {
	FromFQN string
	ToFQN   string
	Kind    string
	File    string
	Line    int
}

// Callers returns symbols that call fqn, up to depth hops.
func (e *Engine) Callers(fqn string, depth int) ([]EdgeResult, error) {
	// TODO: implement
	return nil, nil
}

// Callees returns symbols that fqn calls, up to depth hops.
func (e *Engine) Callees(fqn string, depth int) ([]EdgeResult, error) {
	// TODO: implement
	return nil, nil
}

// Trace finds a path from fromFQN to toFQN using BFS over edges.
func (e *Engine) Trace(fromFQN, toFQN string) ([]string, error) {
	// TODO: implement
	return nil, nil
}

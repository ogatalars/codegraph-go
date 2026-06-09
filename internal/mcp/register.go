package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/ogatalars/codegraph-go/internal/indexer"
	"github.com/ogatalars/codegraph-go/internal/query"
)

// Register adds all codegraph tools to the MCP server.
func Register(s *server.MCPServer, q *query.Engine, idx *indexer.Indexer) {
	// TODO: implement — register search, node, callers, callees, trace, files, status, index
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/truenas/truenas-mcp/mcp"
	"github.com/truenas/truenas-mcp/truenas"
)

// HTTPServer is an HTTP transport for the MCP protocol.
// It follows the Streamable HTTP Transport specification (2025-03-26)
// by accepting POST requests at a single MCP endpoint and returning
// JSON-RPC 2.0 responses.
type HTTPServer struct {
	registry mcp.ToolRegistry
	addr     string
	debug    bool
}

// NewHTTPServer creates a new HTTP MCP server.
func NewHTTPServer(registry mcp.ToolRegistry, addr string, debug bool) *HTTPServer {
	return &HTTPServer{
		registry: registry,
		addr:     addr,
		debug:    debug,
	}
}

// Run starts the HTTP server and blocks until it exits.
func (s *HTTPServer) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/health", s.handleHealth)

	if s.debug {
		log.Printf("Starting HTTP MCP server on %s (MCP endpoint: /mcp)", s.addr)
	}
	return http.ListenAndServe(s.addr, mux)
}

// handleMCP is the single MCP endpoint for Streamable HTTP transport.
// All JSON-RPC messages are POSTed here.
func (s *HTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON-RPC request from the request body
	var req mcp.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if s.debug {
			log.Printf("Failed to parse request body: %v", err)
		}
		s.writeJSONError(w, nil, -32700, fmt.Sprintf("Parse error: %v", err))
		return
	}

	if s.debug {
		log.Printf("[HTTP] Received method: %s (id: %v)", req.Method, req.ID)
	}

	// Route to the same handler logic as stdio
	resp := s.handleRequest(&req)

	// Notifications don't get responses
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	s.writeJSON(w, resp)
}

// handleRequest dispatches an MCP request to the tool registry.
// Mirrors StdioHandler.handleRequest for transport consistency.
func (s *HTTPServer) handleRequest(req *mcp.Request) *mcp.Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		if req.ID != nil {
			return s.createErrorResponse(req.ID, -32601, "Method not found")
		}
		return nil
	}
}

func (s *HTTPServer) handleInitialize(req *mcp.Request) *mcp.Response {
	result := mcp.InitializeResult{
		ProtocolVersion: "2025-03-26",
		ServerInfo: mcp.ServerInfo{
			Name:    "truenas-mcp",
			Version: Version,
		},
		Capabilities: mcp.Capabilities{
			Tools: map[string]interface{}{
				"listChanged": false,
			},
		},
	}
	return &mcp.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *HTTPServer) handleToolsList(req *mcp.Request) *mcp.Response {
	tools := s.registry.ListTools()
	result := mcp.ToolsListResult{
		Tools: tools,
	}
	return &mcp.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *HTTPServer) handleToolsCall(req *mcp.Request) *mcp.Response {
	var params mcp.ToolCallParams
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return s.createErrorResponse(req.ID, -32602, fmt.Sprintf("Invalid params: %v", err))
	}
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return s.createErrorResponse(req.ID, -32602, fmt.Sprintf("Invalid params: %v", err))
	}

	result, err := s.registry.CallTool(params.Name, params.Arguments)
	if err != nil {
		return &mcp.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: mcp.ToolCallResult{
				Content: []mcp.ContentBlock{
					{Type: "text", Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			},
		}
	}
	return &mcp.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: mcp.ToolCallResult{
			Content: []mcp.ContentBlock{
				{Type: "text", Text: result},
			},
		},
	}
}

func (s *HTTPServer) createErrorResponse(id interface{}, code int, message string) *mcp.Response {
	return &mcp.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcp.Error{
			Code:    code,
			Message: message,
		},
	}
}

// handleHealth provides a simple health check endpoint for container orchestration.
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, Version)
}

// writeJSON marshals a response and writes it as JSON.
func (s *HTTPServer) writeJSON(w http.ResponseWriter, resp *mcp.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		if s.debug {
			log.Printf("Failed to marshal response: %v", err)
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if s.debug {
		log.Printf("[HTTP] Response: %s", truenas.RedactJSONForLog(data))
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// writeJSONError writes a JSON-RPC error response.
func (s *HTTPServer) writeJSONError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := s.createErrorResponse(id, code, message)
	s.writeJSON(w, resp)
}
package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// MCPRequest is a JSON-RPC 2.0 request for MCP.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

// MCPResponse is a JSON-RPC 2.0 response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// MCPError represents an MCP error.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Server implements the MCP server for the sandbox API.
type Server struct {
	mu     sync.Mutex
	sessions map[string]*Session
}

// Session represents a sandbox session in the MCP server.
type Session struct {
	ID     string
	Name   string
	Active bool
}

// NewServer creates a new MCP server.
func NewServer() *Server {
	return &Server{
		sessions: make(map[string]*Session),
	}
}

// HandleRequest processes an MCP JSON-RPC request.
func (s *Server) HandleRequest(r io.Reader) (*MCPResponse, error) {
	var req MCPRequest
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			Error:   &MCPError{Code: -32600, Message: "Invalid JSON-RPC request"},
		}, nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params, req.ID)
	case "tools/list":
		return s.handleToolsList(req.ID)
	case "resources/list":
		return s.handleResourcesList(req.ID)
	case "sandbox/create":
		return s.handleSandboxCreate(req.Params, req.ID)
	case "sandbox/exec":
		return s.handleSandboxExec(req.Params, req.ID)
	case "sandbox/destroy":
		return s.handleSandboxDestroy(req.Params, req.ID)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			Error:   &MCPError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
			ID:      req.ID,
		}, nil
	}
}

// HTTPHandler returns an HTTP handler for the MCP server.
func (s *Server) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := s.HandleRequest(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func (s *Server) handleInitialize(params json.RawMessage, id json.RawMessage) (*MCPResponse, error) {
	return &MCPResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    "pi-sandbox-mcp",
				"version": "0.1.0",
			},
			"capabilities": map[string]interface{}{
				"tools":     true,
				"resources": true,
			},
		},
		ID: id,
	}, nil
}

func (s *Server) handleToolsList(id json.RawMessage) (*MCPResponse, error) {
	tools := []map[string]interface{}{
		{
			"name":        "sandbox_create",
			"description": "Create a new sandbox session",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":     map[string]interface{}{"type": "string"},
					"template": map[string]interface{}{"type": "string"},
					"mode":     map[string]interface{}{"type": "string", "enum": []string{"fast", "compat"}},
				},
				"required": []string{"name", "template"},
			},
		},
		{
			"name":        "sandbox_exec",
			"description": "Execute a command in a sandbox session",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sandbox_id": map[string]interface{}{"type": "string"},
					"command":    map[string]interface{}{"type": "string"},
					"timeout_ms": map[string]interface{}{"type": "number"},
				},
				"required": []string{"sandbox_id", "command"},
			},
		},
		{
			"name":        "sandbox_destroy",
			"description": "Destroy a sandbox session",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sandbox_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"sandbox_id"},
			},
		},
		{
			"name":        "sandbox_list",
			"description": "List all sandbox sessions",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"tools": tools,
		},
		ID: id,
	}, nil
}

func (s *Server) handleResourcesList(id json.RawMessage) (*MCPResponse, error) {
	return &MCPResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"resources": []map[string]interface{}{
				{
					"name":        "sandbox_status",
					"description": "Status of sandbox sessions",
					"uri":         "sandbox://status",
				},
			},
		},
		ID: id,
	}, nil
}

func (s *Server) handleSandboxCreate(params json.RawMessage, id json.RawMessage) (*MCPResponse, error) {
	// In production, this would call the daemon API
	// For now, return a stub response
	return &MCPResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"sandbox_id": "stub-" + string(id),
			"status":     "created",
		},
		ID: id,
	}, nil
}

func (s *Server) handleSandboxExec(params json.RawMessage, id json.RawMessage) (*MCPResponse, error) {
	return &MCPResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"exit_code":    0,
			"stdout":       "",
			"stderr":       "",
			"duration_ms":  0,
			"timed_out":    false,
			"truncated":    false,
		},
		ID: id,
	}, nil
}

func (s *Server) handleSandboxDestroy(params json.RawMessage, id json.RawMessage) (*MCPResponse, error) {
	return &MCPResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"status": "destroyed",
		},
		ID: id,
	}, nil
}

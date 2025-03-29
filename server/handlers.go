package server

import (
	"encoding/json"
	"github.com/dentaku7/mcp-go-sdk"
)

// handleInitialize processes the initialize request
func (s *MCPServer) handleInitialize(req *mcp.Request) error {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.sendError(&req.ID, ErrInvalidParams, "Invalid parameters", err.Error())
	}

	// Validate protocol version
	if params.ProtocolVersion == "" {
		return s.sendError(&req.ID, ErrInvalidParams, "Invalid params", "protocolVersion is required")
	}

	// Send initialize result
	result := mcp.InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo: mcp.ServerInfo{
			Name:    "MCP Server",
			Version: "1.0.0",
		},
		Capabilities: mcp.ServerCapabilities{
			Tools: &mcp.ToolsCapability{
				ListChanged: false,
			},
		},
	}

	if err := s.sendResult(&req.ID, result); err != nil {
		return err
	}

	// Send initialized notification
	return s.sendNotification(MethodInitialized, nil)
}

// handleListTools processes the tools/list request
func (s *MCPServer) handleListTools(req *mcp.Request) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]mcp.ToolInfo, len(s.tools))
	for i, tool := range s.tools {
		tools[i] = mcp.ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.Schema(),
		}
	}

	result := mcp.ListToolsResponse{
		Tools: tools,
	}

	return s.sendResult(&req.ID, result)
}

// handleCallTool processes the tools/call request
func (s *MCPServer) handleCallTool(req *mcp.Request) error {
	var params mcp.CallToolRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.sendError(&req.ID, ErrInvalidParams, "Invalid parameters", err.Error())
	}

	s.mu.RLock()
	var tool mcp.Tool
	for _, t := range s.tools {
		if t.Name() == params.Name {
			tool = t
			break
		}
	}
	s.mu.RUnlock()

	if tool == nil {
		return s.sendError(&req.ID, ErrMethodNotFound, "Tool not found", params.Name)
	}

	result, err := tool.Execute(params.Arguments)
	if err != nil {
		return s.sendError(&req.ID, ErrInternal, "Tool execution failed", err.Error())
	}

	return s.sendResult(&req.ID, result)
}

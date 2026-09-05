package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

const protocolVersion = "2025-06-18"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct { Code int `json:"code"`; Message string `json:"message"` }
type toolResult struct { Content []textContent `json:"content"`; IsError bool `json:"isError,omitempty"` }
type textContent struct { Type string `json:"type"`; Text string `json:"text"` }

func serve(in io.Reader, out io.Writer, office *officeClient) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		var request rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			if err := encoder.Encode(rpcResponse{JSONRPC: "2.0", ID: nil, Error: &rpcError{-32700, "parse error"}}); err != nil { return err }
			continue
		}
		// MCP notifications have no id and must not receive a response.
		if len(request.ID) == 0 || string(request.ID) == "null" { continue }
		var id interface{}
		if err := json.Unmarshal(request.ID, &id); err != nil { id = nil }
		response := handleRequest(request, id, office)
		if err := encoder.Encode(response); err != nil { return err }
	}
	return scanner.Err()
}

func handleRequest(request rpcRequest, id interface{}, office *officeClient) rpcResponse {
	if request.JSONRPC != "2.0" { return rpcResponse{JSONRPC:"2.0", ID:id, Error:&rpcError{-32600,"invalid JSON-RPC version"}} }
	switch request.Method {
	case "initialize":
		return rpcResponse{JSONRPC:"2.0", ID:id, Result:map[string]interface{}{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo": map[string]string{"name":"thefloor_mcp", "version":mcpVersionString()},
		}}
	case "tools/list":
		return rpcResponse{JSONRPC:"2.0", ID:id, Result:map[string]interface{}{"tools": toolDefinitions()}}
	case "tools/call":
		var p struct { Name string `json:"name"`; Arguments json.RawMessage `json:"arguments"` }
		if err := json.Unmarshal(request.Params, &p); err != nil { return rpcResponse{JSONRPC:"2.0", ID:id, Error:&rpcError{-32602,"invalid tools/call params"}} }
		text, toolErr := office.call(p.Name, p.Arguments)
		return rpcResponse{JSONRPC:"2.0", ID:id, Result:toolResult{Content:[]textContent{{Type:"text", Text:text}}, IsError:toolErr}}
	default:
		return rpcResponse{JSONRPC:"2.0", ID:id, Error:&rpcError{-32601,"method not found"}}
	}
}

func schema(properties map[string]interface{}, required ...string) map[string]interface{} {
	r := map[string]interface{}{"type":"object", "properties":properties, "additionalProperties":false}
	if len(required) > 0 { r["required"] = required }
	return r
}

func toolDefinitions() []map[string]interface{} {
	str := map[string]interface{}{"type":"string"}
	integer := map[string]interface{}{"type":"integer"}
	return []map[string]interface{}{
		{"name":"plan_present", "description":"Present a plan draft in the live theboringfloor office for member review.", "inputSchema":schema(map[string]interface{}{"text":str}, "text")},
		{"name":"plan_update", "description":"Update a plan draft in the live theboringfloor office.", "inputSchema":schema(map[string]interface{}{"text":str}, "text")},
		{"name":"plan_get_approved", "description":"Get the approved plan from the live office or its on-disk snapshot.", "inputSchema":schema(map[string]interface{}{})},
		{"name":"transcript_read", "description":"Read recent transcript messages from the live office or on-disk snapshot.", "inputSchema":schema(map[string]interface{}{"limit":integer})},
		{"name":"transcript_search", "description":"Search the current project's on-disk transcript.", "inputSchema":schema(map[string]interface{}{"query":str, "limit":integer}, "query")},
		{"name":"office_status", "description":"Report whether the project office is live and its current status.", "inputSchema":schema(map[string]interface{}{})},
	}
}

func decodeArgs(raw json.RawMessage, target interface{}) error {
	if len(raw) == 0 { raw = []byte("{}") }
	if err := json.Unmarshal(raw, target); err != nil { return fmt.Errorf("invalid tool arguments: %w", err) }
	return nil
}

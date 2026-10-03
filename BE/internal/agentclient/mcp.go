package agentclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
)

const ProtocolVersion = "2025-11-25"

func Tools() []any {
	str := map[string]any{"type": "string"}
	return []any{
		map[string]any{"name": "discover", "description": "List finite workspace operations; request schema before action.", "inputSchema": ObjectSchema(map[string]any{"resource": str}), "annotations": map[string]any{"readOnlyHint": true}},
		map[string]any{"name": "schema", "description": "Get one operation's exact input schema.", "inputSchema": ObjectSchema(map[string]any{"operation": str}, "operation"), "annotations": map[string]any{"readOnlyHint": true}},
		map[string]any{"name": "action", "description": "Execute a discovered operation. Compact output; fields/detail opt in. Delivery updates require last read version.", "inputSchema": ObjectSchema(map[string]any{"operation": str, "input": map[string]any{"type": "object"}}, "operation", "input"), "annotations": map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": false}}}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func ServeMCP(ctx context.Context, in io.Reader, out io.Writer, client *Client) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), MaxMessage)
	encoder := json.NewEncoder(out)
	initialized, ready := false, false
	send := func(id json.RawMessage, result any, code int, message string) error {
		r := map[string]any{"jsonrpc": "2.0", "id": id}
		if id == nil {
			r["id"] = nil
		}
		if code != 0 {
			r["error"] = map[string]any{"code": code, "message": message}
		} else {
			r["result"] = result
		}
		return encoder.Encode(r)
	}
	for scanner.Scan() {
		var req rpcRequest
		if json.Unmarshal(scanner.Bytes(), &req) != nil {
			if err := send(nil, nil, -32700, "Parse error"); err != nil {
				return err
			}
			continue
		}
		if req.JSONRPC != "2.0" || req.Method == "" {
			if err := send(nil, nil, -32600, "Invalid Request"); err != nil {
				return err
			}
			continue
		}
		if req.ID != nil {
			var id any
			_ = json.Unmarshal(req.ID, &id)
			switch id.(type) {
			case string, float64:
			default:
				if err := send(nil, nil, -32600, "Invalid request ID"); err != nil {
					return err
				}
				continue
			}
		}
		if req.ID == nil {
			if req.Method == "notifications/initialized" && initialized {
				ready = true
			}
			continue
		}
		var result any
		code, message := 0, ""
		switch req.Method {
		case "initialize":
			var p struct {
				ProtocolVersion string         `json:"protocolVersion"`
				Capabilities    map[string]any `json:"capabilities"`
				ClientInfo      map[string]any `json:"clientInfo"`
			}
			if initialized || json.Unmarshal(req.Params, &p) != nil || p.ProtocolVersion == "" || p.Capabilities == nil || p.ClientInfo == nil || p.ClientInfo["name"] == nil || p.ClientInfo["version"] == nil {
				code, message = -32602, "Invalid initialize parameters"
				break
			}
			version := ProtocolVersion
			if p.ProtocolVersion == "2025-06-18" {
				version = p.ProtocolVersion
			}
			initialized = true
			result = map[string]any{"protocolVersion": version, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "sprintorio", "version": "0.1.0"}}
		case "ping":
			result = map[string]any{}
		default:
			if !ready {
				code, message = -32002, "Initialize and send notifications/initialized first"
				break
			}
			switch req.Method {
			case "tools/list":
				if len(req.Params) > 0 && string(req.Params) != "null" {
					var p map[string]any
					if json.Unmarshal(req.Params, &p) != nil || len(p) > 0 {
						code, message = -32602, "tools/list takes no parameters"
						break
					}
				}
				result = map[string]any{"tools": Tools()}
			case "tools/call":
				var p struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
					Meta      json.RawMessage `json:"_meta"`
				}
				d := json.NewDecoder(bytes.NewReader(req.Params))
				d.DisallowUnknownFields()
				if d.Decode(&p) != nil {
					code, message = -32602, "Invalid tool parameters"
					break
				}
				args, err := DecodeInput(p.Arguments)
				if err != nil {
					code, message = -32602, "Invalid tool arguments"
					break
				}
				var ts map[string]any
				for _, tool := range Tools() {
					t := tool.(map[string]any)
					if t["name"] == p.Name {
						ts = t["inputSchema"].(map[string]any)
					}
				}
				if ts == nil {
					code, message = -32602, "Unknown tool"
					break
				}
				if validate(args, ts) != nil {
					code, message = -32602, "Invalid tool arguments"
					break
				}
				var value any
				switch p.Name {
				case "discover":
					resource, _ := args["resource"].(string)
					value = map[string]any{"operations": Discover(resource)}
				case "schema":
					value, err = Schema(args["operation"].(string))
				case "action":
					if client == nil {
						err = Err("CONFIG_ERROR", "Configure SPRINTORIO_URL, SPRINTORIO_TOKEN and SPRINTORIO_WORKSPACE", 0)
					} else {
						value, err = client.Call(ctx, args["operation"].(string), args["input"].(map[string]any))
					}
				}
				if err != nil {
					e, ok := err.(*Error)
					if !ok {
						e = Err("INTERNAL_ERROR", "Operation failed", 0)
					}
					result = map[string]any{"content": []any{map[string]any{"type": "text", "text": e.Code + ": " + e.Message}}, "structuredContent": map[string]any{"error": e}, "isError": true}
				} else {
					result = map[string]any{"content": []any{}, "structuredContent": map[string]any{"result": value}}
				}
			default:
				code, message = -32601, "Method not found"
			}
		}
		if err := send(req.ID, result, code, message); err != nil {
			return err
		}
	}
	if scanner.Err() != nil {
		return Err("MCP_INPUT_ERROR", "MCP read failed or line exceeds 2 MiB", 0)
	}
	return nil
}

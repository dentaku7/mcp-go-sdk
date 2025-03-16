# Echo Tool Example

A simple example of an MCP tool built with the mcp-go SDK. This tool echoes back any message sent to it, demonstrating basic MCP communication over stdio.

## Running the Example

```bash
go run echo.go
```

## Example Communication

After starting the tool, here's the sequence of messages exchanged:

1. Initialize request:
```json
{"jsonrpc": "2.0", "id": "1", "method": "initialize", "params": {"protocolVersion": "2024-11-05", "clientInfo": {"name": "test-client", "version": "1.0.0"}, "capabilities": {"tools": true}}}
```

2. Initialize response:
```json
{"jsonrpc": "2.0", "result": {"protocolVersion": "2024-11-05", "capabilities": {"tools": {"listChanged": false}}, "serverInfo": {"name": "MCP Server", "version": "1.0.0"}}, "id": "1"}
```

3. Initialized notification:
```json
{"jsonrpc": "2.0", "method": "initialized"}
```

4. Tools list request:
```json
{"jsonrpc": "2.0", "id": "2", "method": "tools/list"}
```

5. Tools list response:
```json
{"jsonrpc": "2.0", "result": {"tools": [{"name": "echo", "description": "A simple echo tool that returns the input message", "inputSchema": {"type": "object", "properties": {"message": {"type": "string", "description": "The message to echo back"}}, "required": ["message"]}}]}, "id": "2"}
```

6. Echo tool call:
```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "echo", "arguments": {"message": "hello from terminal"}}}
```

7. Echo response:
```json
{"jsonrpc": "2.0", "result": {"content": [{"text": "hello from terminal", "type": "text"}], "metadata": {"length": 19}}, "id": 1}
```

The tool can be terminated with Ctrl+C.

## Features Demonstrated

1. Basic MCP tool implementation
2. JSON-RPC 2.0 communication over stdio
3. MCP protocol initialization sequence
4. Tools listing capability
5. Tool invocation and response handling 
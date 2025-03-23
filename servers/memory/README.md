# MCP Memory Server

A Model Context Protocol (MCP) server providing a persistent in-memory knowledge graph. Stores entities, relations, and observations in a JSON file with tools to manipulate and query the graph.

## Installation

```bash
go install github.com/dentaku7/mcp-go-sdk/servers/memory@latest
```

## Usage

Start the server:

```bash
memory-server [--path memory.json]
```

Configuration options:
- `--path`: Path to the memory file (defaults to `memory.json`)
- Environment variable `MEMORY_FILE_PATH`: Alternative way to specify memory file path

## Tools

### create_entities
```json
{
  "entities": [
    {
      "id": "string",
      "type": "string", 
      "name": "string",
      "description": "string (optional)",
      "metadata": {} (optional)
    }
  ]
}
```

### create_relations
```json
{
  "relations": [
    {
      "id": "string",
      "type": "string",
      "source": "string",
      "target": "string", 
      "description": "string (optional)",
      "metadata": {} (optional)
    }
  ]
}
```

### add_observations
```json
{
  "observations": [
    {
      "id": "string",
      "entity_id": "string",
      "type": "string",
      "content": "string",
      "description": "string (optional)",
      "metadata": {} (optional)
    }
  ]
}
```

### delete_entities
```json
{
  "ids": ["string"]
}
```

### delete_relations
```json
{
  "ids": ["string"]
}
```

### delete_observations
```json
{
  "ids": ["string"]
}
```

### read_graph
```json
{}
```

### search_nodes
```json
{
  "type": "string (optional)",
  "metadata": {} (optional)
}
```

### open_nodes
```json
{
  "ids": ["string"]
}
```

## Development

1. Clone the repository:
   ```bash
   git clone https://github.com/dentaku7/mcp-go-sdk.git
   ```
2. Navigate to memory server:
   ```bash
   cd mcp-go-sdk/servers/memory
   ```
3. Build and run:
   ```bash
   go build ./...
   ./mcp-memory --path memory.json
   ```

## License

MIT 
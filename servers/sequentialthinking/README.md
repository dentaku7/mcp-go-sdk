# Sequential Thinking Tool

A Model Communication Protocol (MCP) tool for dynamic and reflective problem-solving through sequential thoughts. This tool helps analyze problems through a flexible thinking process that can adapt and evolve.

## Features

- Break down complex problems into steps
- Support for planning and design with room for revision
- Analysis with course correction capabilities
- Handle problems where the full scope might not be clear initially
- Maintain context over multiple steps
- Branch and revise thoughts as understanding deepens
- Generate and verify solution hypotheses

## Prerequisites

Requires Go 1.21 or later.

## Usage

The tool is designed to be used with the MCP protocol. To use it with Cursor IDE:

1. Create a `.cursor/mcp.json` file in your project root:

```json
{
    "mcpServers": {
        "sequentialthinking": {
            "command": "sequentialthinking",
            "args": [],
            "env": {}
        }
    }
}
```

2. Restart Cursor IDE to load the new MCP configuration.

3. The tool will be available for use in your AI conversations.

### Thought Structure

A thought requires the following fields:
- `thought`: The actual thought content (string)
- `thoughtNumber`: Position in the sequence (integer ≥ 1)
- `totalThoughts`: Total expected thoughts (integer ≥ 1)
- `nextThoughtNeeded`: Whether another thought is needed (boolean)

Optional fields:
- `isRevision`: Whether this revises a previous thought (boolean)
- `revisesThought`: Which thought is being revised (integer ≥ 1)
- `branchFromThought`: Parent thought for branching (integer ≥ 1)
- `branchId`: Branch identifier (string)
- `needsMoreThoughts`: If more thoughts are needed (boolean)

### Example

```json
{
  "thought": "Initial analysis of the problem",
  "thoughtNumber": 1,
  "totalThoughts": 3,
  "nextThoughtNeeded": true
}
```

## Development

### Project Structure

```
.
├── cmd/
│   └── sequentialthinking/  # Command line tool
│       └── main.go         # Entry point
├── pkg/
│   └── seqthinking/        # Core tool implementation
│       ├── seqthinking.go
│       └── seqthinking_test.go
└── README.md
```

The `cmd/` directory contains the executable entry points, while `pkg/` contains the reusable package code.

### Running Tests

```bash
go test ./...
```

## Protocol

This tool implements the Model Communication Protocol (MCP) for standardized communication between AI models and tools. The protocol version supported is 2024-11-05.

### Message Format

Responses follow this format:
```json
{
  "content": [
    {
      "type": "text",
      "text": "formatted thought"
    }
  ],
  "metadata": {
    "thoughtNumber": 1,
    "totalThoughts": 3,
    "nextThoughtNeeded": true,
    "branches": ["branch-id"],
    "thoughtHistoryLength": 1
  },
  "isError": false
}
``` 
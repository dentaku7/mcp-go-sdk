module mcp-duckdb

go 1.21

require (
	github.com/marcboeker/go-duckdb v1.5.6
	mcp-go-sdk v0.0.0
)

require github.com/mitchellh/mapstructure v1.5.0 // indirect

replace mcp-go-sdk => ../../mcp-go-sdk

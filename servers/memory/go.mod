module mcp-memory

go 1.21

require mcp-go-sdk v0.0.0

require (
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace mcp-go-sdk => ../../mcp-go-sdk

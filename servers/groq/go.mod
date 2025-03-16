module groq

go 1.23.0

toolchain go1.24.0

require (
	github.com/joho/godotenv v1.5.1
	github.com/openai/openai-go v0.1.0-alpha.62
	golang.org/x/time v0.11.0
	mcp-go-sdk v0.0.0-00010101000000-000000000000
)

require (
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)

replace mcp-go-sdk => ../../mcp-go-sdk

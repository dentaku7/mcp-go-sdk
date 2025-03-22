package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"golang.org/x/time/rate"
)

// ToolParams represents the parameters for the Groq tool
type ToolParams struct {
	Question    string   `json:"question"`
	Context     string   `json:"context"`
	Model       *string  `json:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

// Config holds the tool configuration
type Config struct {
	APIKey      string
	Model       string
	Temperature float64
}

// GroqTool implements the MCP Tool interface
type GroqTool struct {
	client    *openai.Client
	rateLimit *rate.Limiter
	config    Config
}

// NewGroqTool creates a new instance of the Groq tool
func NewGroqTool(config Config) *GroqTool {
	// Configure rate limiting
	requestsPerMinute := 20
	if rpmStr := os.Getenv("MAX_REQUESTS_PER_MINUTE"); rpmStr != "" {
		if rpm, err := strconv.Atoi(rpmStr); err == nil {
			requestsPerMinute = rpm
		}
	}

	// Create OpenAI client with Groq configuration
	baseURL := os.Getenv("GROQ_API_BASE")
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1/"
	}

	// Validate API key
	if config.APIKey == "" {
		panic("API key is required")
	}

	client := openai.NewClient(
		option.WithAPIKey(config.APIKey),
		option.WithBaseURL(baseURL),
	)

	return &GroqTool{
		client:    client,
		rateLimit: rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), 1),
		config:    config,
	}
}

// Name returns the name of the tool
func (t *GroqTool) Name() string {
	return "ask_groq"
}

// Description returns a description of the tool
func (t *GroqTool) Description() string {
	return `Ask Groq is your trusted companion for gaining additional perspectives and validating your thinking process, powered by a model with a 128K token context window. 
This means it can process and analyze large amounts of context - entire codebases, long SQL queries, or detailed documentation.

Use this tool when you want to:
1. Validate your proposed solutions before presenting them
2. Get a fresh perspective on your technical reasoning
3. Explore alternative approaches you might not have considered
4. Quickly validate your initial thoughts on a problem
5. Challenge your assumptions and expand your thinking

Parameters:
- question: What you'd like a second opinion on (required)
- context: FULL context to help understand the question better (required). The model has a large 128K token context window, so don't hesitate to provide:
  * COMPLETE code snippets, not just fragments
  * ENTIRE SQL queries with schema definitions
  * FULL error messages and stack traces
  * ALL relevant documentation
  * COMPLETE conversation history
  * DETAILED system state or environment details
  * ANY and ALL information that could be relevant

Example usage:
1. SQL Query Review:
   {
     "question": "Can this query be optimized?",
     "context": "FULL table schemas:\n[complete CREATE TABLE statements]\n\nCurrent query:\n[entire SQL query]\n\nExisting indexes:\n[all index definitions]\n\nTypical data volumes:\n[table sizes and growth patterns]"
   }

2. Code Review:
   {
     "question": "What improvements can we make to this implementation?",
     "context": "Project structure:\n[directory tree]\n\nRelevant files:\n[complete file contents, not just snippets]\n\nDependencies:\n[full package.json/go.mod]\n\nCurrent implementation:\n[entire implementation including imports and related files]"
   }

3. Architecture Review:
   {
     "question": "Is this the right approach?",
     "context": "Current architecture:\n[complete system diagram]\n\nRequirements:\n[full requirements doc]\n\nConstraints:\n[all technical and business constraints]\n\nExisting components:\n[detailed description of all related systems]"
   }

Remember: This model can handle extensive context (128K tokens), so don't summarize or truncate your context. 
The more complete information you provide, the more accurate and helpful the response will be. 
Include full code, complete queries, entire error messages, and comprehensive documentation whenever possible.`
}

// Schema returns the JSON schema for the tool's parameters
func (t *GroqTool) Schema() json.RawMessage {
	schema := `{
		"type": "object",
		"properties": {
			"question": {
				"type": "string",
				"description": "What you'd like a second opinion on"
			},
			"context": {
				"type": "string",
				"description": "Context to help to understand the question better. Include relevant background information, code snippets, or documentation."
			}
		},
		"required": ["question", "context"]
	}`
	return json.RawMessage(schema)
}

// Execute runs the tool with the given parameters
func (t *GroqTool) Execute(params json.RawMessage) (interface{}, error) {
	// Parse parameters
	var toolParams ToolParams
	if err := json.Unmarshal(params, &toolParams); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %v", err)
	}

	// Validate required parameters
	if toolParams.Question == "" {
		return nil, fmt.Errorf("empty question")
	}
	if toolParams.Context == "" {
		return nil, fmt.Errorf("context is required")
	}

	// Apply rate limiting with context
	ctx := context.Background()
	if err := t.rateLimit.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %v", err)
	}

	// Use config values or override with provided parameters
	model := t.config.Model
	if toolParams.Model != nil {
		model = *toolParams.Model
	}

	temperature := t.config.Temperature
	if toolParams.Temperature != nil {
		temperature = *toolParams.Temperature
		if temperature < 0 || temperature > 1 {
			return nil, fmt.Errorf("temperature must be between 0 and 1")
		}
		// Handle Groq's special case for temperature=0
		if temperature == 0 {
			temperature = 1e-8
		}
	}

	// Prepare messages array with default system role
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.ChatCompletionMessage{
			Role:    "system",
			Content: "You are a helpful expert assistant who provides clear, practical, and well-reasoned answers. You excel at understanding context and providing targeted responses. If you feel the provided context is insufficient to give a complete and accurate answer, don't hesitate to ask for specific additional information that would help you provide a better response.",
		},
		openai.ChatCompletionMessage{
			Role:    "user",
			Content: toolParams.Context,
		},
		openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: "I understand the context you've provided. I'll use this information to answer your question.",
		},
		openai.ChatCompletionMessage{
			Role:    "user",
			Content: toolParams.Question,
		},
	}

	// Create chat completion request
	resp, err := t.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages:    openai.F(messages),
		Model:       openai.F(model),
		Temperature: openai.F(temperature),
	})
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %v", err)
	}

	// Return formatted response
	return map[string]interface{}{
		"isError": false,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": resp.Choices[0].Message.Content,
			},
		},
	}, nil
}

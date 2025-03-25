package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
	return `Ask Groq is your trusted companion for gaining additional perspectives and validating your thinking process, powered by DeepSeek-R1 with a 128K token context window.

IMPORTANT USAGE GUIDELINES:
1. Temperature Setting (Critical):
   - RECOMMENDED: 0.6 (default, best for most use cases)
   - VALID RANGE: 0.5-0.7
   - WARNING: Values outside this range may cause repetitive or incoherent outputs
   - SPECIAL CASES:
     * Technical/Math problems: Use 0.5 for more precise outputs
     * General discussion: Use 0.7 for more varied responses
     Note: Higher temperatures increase token usage and costs

2. Response Processing:
   - The model performs thorough step-by-step reasoning internally
   - Only the final, refined answer is presented to you
   - Mathematical solutions will show final answers in \boxed{} format
   - Technical problems will show clear, structured conclusions
   - Complex solutions will present concise, actionable insights

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
- temperature: Controls response precision (optional, default: 0.6)
  WARNING: Stay within 0.5-0.7 range for optimal results

Example usage:
1. Mathematical Problem:
   {
     "question": "Solve the quadratic equation: x² + 5x + 6 = 0",
     "context": "Please solve this step by step and provide the final answer in a boxed format.",
     "temperature": 0.5
   }
   You'll receive:
   \boxed{x = -3 \text{ or } x = -2}

2. Code Review:
   {
     "question": "Review this implementation for potential improvements",
     "context": "Project structure:\n[directory tree]\n\nRelevant files:\n[complete file contents]\n\nCurrent implementation:\n[entire implementation]",
     "temperature": 0.6
   }
   You'll receive:
   Recommended improvements:
   1. Add error handling for edge cases
   2. Optimize database queries
   3. Improve code documentation
   [Detailed explanation for each point]

3. Architecture Review:
   {
     "question": "Evaluate this system design",
     "context": "Current architecture:\n[complete system diagram]\n\nRequirements:\n[full requirements]\n\nConstraints:\n[all constraints]",
     "temperature": 0.6
   }
   You'll receive:
   Architecture Assessment:
   1. Scalability considerations
   2. Security recommendations
   3. Performance optimization opportunities
   [Detailed analysis for each aspect]

Remember: 
- Always provide complete context - the model can handle 128K tokens
- The model performs thorough internal analysis before providing answers
- Mathematical solutions will be presented in \boxed{} format
- Technical answers will be clear and actionable
- Stay within recommended temperature range (0.5-0.7)`
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
			},
			"temperature": {
				"type": "number",
				"description": "Controls response creativity (0.0-1.5). Use 0.0 for precise outputs, 0.6 (default) for balanced responses, 1.5 for creative outputs.",
				"minimum": 0,
				"maximum": 1.5
			}
		},
		"required": ["question", "context"]
	}`
	return json.RawMessage(schema)
}

// extractFinalAnswer removes the thinking process and returns only the final answer
func extractFinalAnswer(response string) string {
	// Find the end of thinking section
	thinkEnd := strings.Index(response, "</think>")
	if thinkEnd == -1 {
		// If no think tags found, return original response
		return response
	}

	// Get everything after </think>
	answer := strings.TrimSpace(response[thinkEnd+8:]) // 8 is length of "</think>"
	if answer == "" {
		// If no content after </think>, return original response
		return response
	}

	return answer
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
		if temperature < 0 || temperature > 1.5 {
			return nil, fmt.Errorf("temperature must be between 0 and 1.5")
		}
		if temperature < 0.5 || temperature > 0.7 {
			log.Printf("WARNING: Temperature %.1f is outside recommended range (0.5-0.7). This may cause unexpected behavior.", temperature)
		}
		// Handle special case for temperature=0
		if temperature == 0 {
			temperature = 1e-8
		}
	}

	// Prepare messages array with instructions and context
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.ChatCompletionMessage{
			Role:    "user",
			Content: fmt.Sprintf("%s\n\nInstructions:\n- Think step by step\n- Start your thinking with <think>\n- Show your reasoning process\n- For mathematical problems, put final answers in \\boxed{}\n\n%s", toolParams.Context, toolParams.Question),
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

	// Process the response to extract final answer
	finalAnswer := extractFinalAnswer(resp.Choices[0].Message.Content)

	// Return formatted response
	return map[string]interface{}{
		"isError": false,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": finalAnswer,
			},
		},
	}, nil
}

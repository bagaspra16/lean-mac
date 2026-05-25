package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

const groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"

// Message is the OpenAI-compatible chat message shape.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool is what we advertise to the model.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunctionDef `json:"function"`
}

type ToolFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Client speaks the Groq chat API and rotates keys on rate limits / errors.
type Client struct {
	keys   []string
	model  string
	idx    atomic.Uint32
	http   *http.Client
}

func NewClient(keys []string, model string) *Client {
	return &Client{
		keys:  keys,
		model: model,
		http:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Chat sends messages + tool definitions, returns the next assistant message.
// On 429 / 5xx, it rotates to the next key and retries (up to len(keys) total
// attempts).
func (c *Client) Chat(ctx context.Context, msgs []Message, tools []Tool) (Message, error) {
	if len(c.keys) == 0 {
		return Message{}, errors.New("no Groq API keys configured")
	}
	req := chatRequest{
		Model:       c.model,
		Messages:    msgs,
		Tools:       tools,
		Temperature: 0.2,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, err
	}

	var lastErr error
	for attempt := 0; attempt < len(c.keys); attempt++ {
		key := c.keys[int(c.idx.Load())%len(c.keys)]
		resp, err := c.do(ctx, key, body)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// rotate on any error; if it's a permanent 4xx (auth, bad request),
		// the next key will likely fail too, but we let the loop terminate naturally.
		c.idx.Add(1)
	}
	return Message{}, fmt.Errorf("all %d keys exhausted: %w", len(c.keys), lastErr)
}

func (c *Client) do(ctx context.Context, key string, body []byte) (Message, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", groqEndpoint, bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var cr chatResponse
		_ = json.Unmarshal(data, &cr)
		msg := resp.Status
		if cr.Error != nil {
			msg = cr.Error.Message
		}
		return Message{}, fmt.Errorf("groq %d: %s", resp.StatusCode, msg)
	}
	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return Message{}, fmt.Errorf("decode: %w", err)
	}
	if len(cr.Choices) == 0 {
		return Message{}, errors.New("no choices in response")
	}
	return cr.Choices[0].Message, nil
}

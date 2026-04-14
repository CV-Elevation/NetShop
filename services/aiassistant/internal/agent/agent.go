package agent

import (
	"context"
	"errors"
	"log"
	"netshop/services/aiassistant/shared"
	"netshop/services/aiassistant/tool"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type StreamEvent struct {
	Type        string
	Delta       string
	ToolName    string
	ToolStatus  string
	ToolSummary string
	Done        bool
}

type Agent struct {
	systemPrompt string
	model        string
	client       openai.Client
	messages     []openai.ChatCompletionMessageParamUnion
	tools        map[tool.AgentTool]tool.Tool
}

func NewAgent(modelConf shared.ModelConfig, systemPrompt string, tools []tool.Tool) *Agent {
	a := Agent{
		systemPrompt: systemPrompt,
		model:        modelConf.Model,
		client:       openai.NewClient(option.WithBaseURL(modelConf.BaseURL), option.WithAPIKey(modelConf.ApiKey)),
		tools:        make(map[tool.AgentTool]tool.Tool),
		messages:     make([]openai.ChatCompletionMessageParamUnion, 0),
	}
	for _, t := range tools {
		a.tools[t.ToolName()] = t
	}
	a.messages = append(a.messages, openai.SystemMessage(systemPrompt))
	return &a
}

func (a *Agent) execute(ctx context.Context, toolName string, argumentsInJSON string) (string, error) {
	t, ok := a.tools[tool.AgentTool(toolName)]
	if !ok {
		return "", errors.New("tool not found")
	}
	return t.Execute(ctx, argumentsInJSON)
}

func (a *Agent) buildParams(messages []openai.ChatCompletionMessageParamUnion) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    a.model,
		Messages: messages,
		Tools:    make([]openai.ChatCompletionToolUnionParam, 0, len(a.tools)),
	}

	for _, t := range a.tools {
		params.Tools = append(params.Tools, t.Info())
	}

	return params
}

func (a *Agent) runOnce(ctx context.Context, query string, emit func(StreamEvent) error) (string, error) {
	a.messages = append(a.messages, openai.UserMessage(query))
	var result string

	for {
		params := a.buildParams(a.messages)

		if emit == nil {
			log.Printf("calling llm model %s...", a.model)

			resp, err := a.client.Chat.Completions.New(ctx, params)
			if err != nil {
				log.Fatalf("failed to send a new completion request: %v", err)
				return "", err
			}
			if len(resp.Choices) == 0 {
				log.Printf("no choices returned, resp: %v", resp)
				return "", nil
			}

			message := resp.Choices[0].Message
			a.messages = append(a.messages, message.ToParam())

			if len(message.ToolCalls) == 0 {
				result = message.Content
				break
			}

			for _, toolCall := range message.ToolCalls {
				toolResult, err := a.execute(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
				if err != nil {
					toolResult = err.Error()
				}
				log.Printf("tool call %s, arguments %s, error: %v", toolCall.Function.Name, toolCall.Function.Arguments, err)
				a.messages = append(a.messages, openai.ToolMessage(toolResult, toolCall.ID))
			}
			continue
		}

		log.Printf("calling llm model %s in streaming mode...", a.model)
		stream := a.client.Chat.Completions.NewStreaming(ctx, params)
		acc := openai.ChatCompletionAccumulator{}

		for stream.Next() {
			chunk := stream.Current()
			if !acc.AddChunk(chunk) {
				return "", errors.New("failed to accumulate streamed completion chunk")
			}

			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					if err := emit(StreamEvent{Type: "text", Delta: delta}); err != nil {
						return "", err
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			return "", err
		}

		if len(acc.Choices) == 0 {
			return "", nil
		}

		message := acc.Choices[0].Message
		a.messages = append(a.messages, message.ToParam())

		if len(message.ToolCalls) == 0 {
			result = message.Content
			if err := emit(StreamEvent{Type: "done", Done: true}); err != nil {
				return "", err
			}
			break
		}

		for _, toolCall := range message.ToolCalls {
			summary := strings.TrimSpace(toolCall.Function.Name)
			if summary == "" {
				summary = "tool call"
			}
			if err := emit(StreamEvent{
				Type:        "tool_status",
				ToolName:    toolCall.Function.Name,
				ToolStatus:  "running",
				ToolSummary: summary,
			}); err != nil {
				return "", err
			}

			toolResult, err := a.execute(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
			toolStatus := "done"
			if err != nil {
				toolStatus = "error"
				toolResult = err.Error()
			}

			if err := emit(StreamEvent{
				Type:        "tool_status",
				ToolName:    toolCall.Function.Name,
				ToolStatus:  toolStatus,
				ToolSummary: toolResult,
			}); err != nil {
				return "", err
			}

			log.Printf("tool call %s, arguments %s, error: %v", toolCall.Function.Name, toolCall.Function.Arguments, err)
			a.messages = append(a.messages, openai.ToolMessage(toolResult, toolCall.ID))
		}
	}

	return result, nil
}

// Run 提供对于单次用户请求 query 的 tool loop，返回本轮结果的输出。Run 会保持当前对话历史，不同主题的对话轮次应该初始化多个 Agent 实例运行。
func (a *Agent) Run(ctx context.Context, query string) (string, error) {
	return a.runOnce(ctx, query, nil)
}

func (a *Agent) Stream(ctx context.Context, query string, emit func(StreamEvent) error) error {
	_, err := a.runOnce(ctx, query, emit)
	return err
}

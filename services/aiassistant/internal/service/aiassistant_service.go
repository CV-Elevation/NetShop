package service

import (
	"context"
	"log"
	"netshop/services/aiassistant/internal/agent"

	aiassistantpb "kuoz/netshop/platform/shared/proto/aiassistant"
)

type AIAssistantService struct {
	agent *agent.Agent
}

func NewAIAssistantService(agent *agent.Agent) *AIAssistantService {
	return &AIAssistantService{agent: agent}
}

func (s *AIAssistantService) Chat(ctx context.Context, req *aiassistantpb.ChatRequest) (*aiassistantpb.ChatResponse, error) {
	msg := req.Message
	result, err := s.agent.Run(ctx, msg)
	if err != nil {
		log.Printf("agent run error: %v", err)
		return nil, err
	}
	return &aiassistantpb.ChatResponse{
		Text: result,
	}, nil
}

func (s *AIAssistantService) ChatStream(ctx context.Context, req *aiassistantpb.ChatRequest, send func(*aiassistantpb.ChatChunk) error) error {
	return s.agent.Stream(ctx, req.GetMessage(), func(event agent.StreamEvent) error {
		chunk := &aiassistantpb.ChatChunk{ChunkType: event.Type}
		switch event.Type {
		case "text":
			chunk.Delta = event.Delta
		case "tool_status":
			chunk.ToolCall = &aiassistantpb.ToolCall{
				ToolName: event.ToolName,
				Status:   event.ToolStatus,
				Summary:  event.ToolSummary,
			}
		case "done":
			chunk.Done = true
		default:
			chunk.Delta = event.Delta
		}

		return send(chunk)
	})
}

package client

import (
	"context"
	"io"

	aiassistantpb "kuoz/netshop/platform/shared/proto/aiassistant"
	commonpb "kuoz/netshop/platform/shared/proto/common"
)

type AIAssistantServiceClient struct {
	grpcClient aiassistantpb.AiAssistantServiceClient
}

type ChatRequest struct {
	SessionID string
	UserID    string
	Message   string
	History   []*aiassistantpb.Message
}

type ChatResponse struct {
	Text      string
	ToolCalls []*aiassistantpb.ToolCall
	Products  []*commonpb.Product
}

func NewAIAssistantServiceClient(grpcClient aiassistantpb.AiAssistantServiceClient) *AIAssistantServiceClient {
	return &AIAssistantServiceClient{grpcClient: grpcClient}
}

func (c *AIAssistantServiceClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	resp, err := c.grpcClient.Chat(ctx, &aiassistantpb.ChatRequest{
		SessionId: req.SessionID,
		UserId:    req.UserID,
		Message:   req.Message,
		History:   req.History,
	})
	if err != nil {
		return ChatResponse{}, err
	}

	return ChatResponse{
		Text:      resp.GetText(),
		ToolCalls: resp.GetToolCalls(),
		Products:  resp.GetProducts(),
	}, nil
}

func (c *AIAssistantServiceClient) ChatStream(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	stream, err := c.grpcClient.ChatStream(ctx, &aiassistantpb.ChatRequest{
		SessionId: req.SessionID,
		UserId:    req.UserID,
		Message:   req.Message,
		History:   req.History,
	})
	if err != nil {
		return ChatResponse{}, err
	}

	result := ChatResponse{
		ToolCalls: make([]*aiassistantpb.ToolCall, 0),
		Products:  make([]*commonpb.Product, 0),
	}

	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return ChatResponse{}, recvErr
		}

		switch chunk.GetChunkType() {
		case "text":
			result.Text += chunk.GetDelta()
		case "tool_status":
			if call := chunk.GetToolCall(); call != nil {
				result.ToolCalls = append(result.ToolCalls, call)
			}
		case "products":
			result.Products = chunk.GetProducts()
		case "done":
			if chunk.GetDone() {
				return result, nil
			}
		default:
			if chunk.GetDelta() != "" {
				result.Text += chunk.GetDelta()
			}
		}
	}

	return result, nil
}

func (c *AIAssistantServiceClient) StreamChat(ctx context.Context, req ChatRequest, onChunk func(*aiassistantpb.ChatChunk) error) error {
	if onChunk == nil {
		return nil
	}

	stream, err := c.grpcClient.ChatStream(ctx, &aiassistantpb.ChatRequest{
		SessionId: req.SessionID,
		UserId:    req.UserID,
		Message:   req.Message,
		History:   req.History,
	})
	if err != nil {
		return err
	}

	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			return nil
		}
		if recvErr != nil {
			return recvErr
		}
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
}

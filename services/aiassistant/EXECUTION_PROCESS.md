# AIAssistant 执行过程记录

## 背景

本次处理的目标是补全 AI Assistant 的流式聊天能力，并梳理一次从报错到修复的完整执行过程。

## 问题现象

1. 运行根目录脚本时，`./run.sh` 提示 `permission denied`。
2. `services/aiassistant/internal/service/aiassistant_service.go` 中的 `ChatStream` 还是占位实现，直接返回 `nil`。
3. 前端 [services/frontend/internal/handler/web.go](../frontend/internal/handler/web.go) 已经按 NDJSON 的方式消费流式事件，因此后端必须真正输出流事件。

## 定位过程

1. 先检查了 [services/aiassistant/internal/service/aiassistant_service.go](internal/service/aiassistant_service.go) 中的 `ChatStream` 实现，确认它目前只是 `return nil`。
2. 再查看 [services/aiassistant/internal/agent/agent.go](internal/agent/agent.go)，发现 `Agent` 只有非流式的 `Run`，没有流式接口。
3. 进一步查看 proto 定义 [shared/protos/aiassistant.proto](../../shared/protos/aiassistant.proto)，确认流式接口使用 `ChatChunk`，支持的事件类型包括 `text`、`tool_status`、`products` 和 `done`。
4. 检查了 OpenAI Go SDK 的实现，确认可以使用 `Chat.Completions.NewStreaming(...)` 获取流式 chunk，并用 `ChatCompletionAccumulator` 累积工具调用和最终消息。

## 实现方案

### 1. 在 agent 层增加流式执行能力

修改 [services/aiassistant/internal/agent/agent.go](internal/agent/agent.go)：

- 新增 `StreamEvent`，用于在 agent 层描述流式事件。
- 新增 `buildParams(...)`，把模型参数和 tools 统一构造出来，避免 `Run` 和 `Stream` 重复代码。
- 新增 `runOnce(...)`，把一次完整的 tool loop 抽出来，支持两种模式：
  - 非流式：继续走原来的 `Run`
  - 流式：通过 `NewStreaming(...)` 逐步读取模型输出
- 在流式模式下：
  - 收到文本增量时发出 `text`
  - 收到工具调用时发出 `tool_status`
  - 工具执行完成后再发出一次 `tool_status`
  - 最终完成时发出 `done`

### 2. 在 service 层把 agent 事件映射成 proto chunk

修改 [services/aiassistant/internal/service/aiassistant_service.go](internal/service/aiassistant_service.go)：

- `ChatStream(...)` 不再返回空实现。
- 直接调用 `agent.Stream(...)`。
- 将 `StreamEvent` 映射为 `aiassistantpb.ChatChunk`：
  - `text` -> `Delta`
  - `tool_status` -> `ToolCall`
  - `done` -> `Done`

这样前端无需改动，仍然可以按 NDJSON 一行一事件的方式消费结果。

## 验证结果

1. 对 [services/aiassistant/internal/agent/agent.go](internal/agent/agent.go) 和 [services/aiassistant/internal/service/aiassistant_service.go](internal/service/aiassistant_service.go) 做了静态检查，确认没有编译错误。
2. 对修改后的 Go 文件执行了 `gofmt`，保证格式一致。

## 备注

当前实现已经补齐了文本流和工具状态流。如果后续需要让 `products` 事件也稳定输出，需要再把工具返回值结构化，并在 agent/service 之间显式传递商品列表。
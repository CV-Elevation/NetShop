package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ── Ollama 客户端 ─────────────────────────────────────────────

type IntentClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewIntentClient(baseURL, model string) *IntentClient {
	return &IntentClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type ollamaRequest struct {
	Model     string          `json:"model"`
	Messages  []ollamaMessage `json:"messages"`
	Think     bool            `json:"think"`
	Stream    bool            `json:"stream"`
	KeepAlive string          `json:"keep_alive"`
	Options   ollamaOptions   `json:"options"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

func (c *IntentClient) Complete(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model: c.model,
		Messages: []ollamaMessage{
			{Role: "user", Content: prompt},
		},
		Think:     false,
		Stream:    false,
		KeepAlive: "30m",
		Options: ollamaOptions{
			Temperature: 0.1,
			NumPredict:  64,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/api/chat",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(data, &ollamaResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w, 原始输出: %s", err, string(data))
	}

	return strings.TrimSpace(ollamaResp.Message.Content), nil
}

// ── 意图识别 Prompt ───────────────────────────────────────────

const intentPrompt = `你是意图识别器。分析用户消息，给出用户消息中包含的意图。

意图类型：
- product_search：用户表现出购买意向、寻找特定商品或请求推荐。
- customer_service：涉及订单、退换货、物流、支付、补差价、质量投诉等具体“售后或交易流程”问题。
- query_product_performance：询问商品性能、参数、使用体验等问题，表现出对商品的兴趣但不直接表达购买意向。
- chitchat：包含问候、无意义闲聊。
规则：
- 判断是否包含查询商品的语义,如推荐、找找、有没有、想买、适合露营用的等,如果有则添加 product_search 意图
- 判断是否包含客服相关的语义,如订单、退货、物流、支付、售后、丢失包裹、购买的商品有质量问题等,如果有则添加 customer_service 意图
- 判断是否包含商品性能相关的语义，如询问商品参数、性能、使用体验等，但没有直接表达购买意向的，如果有则添加 query_product_performance 意图
- 判断是否包含问候、闲聊等无购买或客服意图的语义，如果有则添加 chitchat 意图
- 核心原则： 请独立判断用户消息的每一个子句。如果一句话前半部分在抱怨质量（客服），后半部分在求推荐（搜索），必须同时返回两个意图。

输出格式示例（只输出 JSON，不要有多余文字）：
{
  "intents": ["product_search", "customer_service"]
}
所有意图都不存在时输出：
{
  "intents": []
}
用户消息：`

// ── 测试用例 ──────────────────────────────────────────────────

type TestCase struct {
	Input           string   // 用户输入
	ExpectedIntents []string // 期望出现的意图（置信度应该 > 0.5）
}

var testCases = []TestCase{
	// product_search 单意图
	// {Input: "有没有防水的登山鞋", ExpectedIntents: []string{"product_search"}},
	// {Input: "推荐一款适合跑步的鞋子", ExpectedIntents: []string{"product_search"}},
	// {Input: "我想买一双耐克运动鞋", ExpectedIntents: []string{"product_search"}},
	// {Input: "有没有红色的连衣裙", ExpectedIntents: []string{"product_search"}},
	// {Input: "200块以内的蓝牙耳机有哪些", ExpectedIntents: []string{"product_search"}},
	// {Input: "帮我找找保暖内衣", ExpectedIntents: []string{"product_search"}},
	// {Input: "有没有儿童书包推荐", ExpectedIntents: []string{"product_search"}},
	// {Input: "想看轻薄一点的笔记本电脑", ExpectedIntents: []string{"product_search"}},
	// {Input: "给我推荐适合露营用的手电筒", ExpectedIntents: []string{"product_search"}},
	// {Input: "有没有支持降噪的头戴式耳机", ExpectedIntents: []string{"product_search"}},
	// {Input: "比较一下你们家两款空气炸锅", ExpectedIntents: []string{"product_search"}},
	// {Input: "我预算500以内买个机械键盘", ExpectedIntents: []string{"product_search"}},

	// // customer_service 单意图
	// {Input: "我的订单还没发货怎么办", ExpectedIntents: []string{"customer_service"}},
	// {Input: "退货流程是什么", ExpectedIntents: []string{"customer_service"}},
	// {Input: "我想申请退款", ExpectedIntents: []string{"customer_service"}},
	// {Input: "快递一直显示在途怎么回事", ExpectedIntents: []string{"customer_service"}},
	// {Input: "收到的商品有破损", ExpectedIntents: []string{"customer_service"}},
	// {Input: "发票怎么开", ExpectedIntents: []string{"customer_service"}},
	// {Input: "我的账号被封了", ExpectedIntents: []string{"customer_service"}},
	// {Input: "优惠券用不了", ExpectedIntents: []string{"customer_service"}},
	// {Input: "我支付成功了但订单还是未支付", ExpectedIntents: []string{"customer_service"}},
	// {Input: "可以帮我催一下快递吗", ExpectedIntents: []string{"customer_service"}},
	// {Input: "这单我想改收货地址", ExpectedIntents: []string{"customer_service"}},
	// {Input: "我买到假货了怎么处理", ExpectedIntents: []string{"customer_service"}},
	// {Input: "售后电话是多少", ExpectedIntents: []string{"customer_service"}},

	// // chitchat 单意图
	// {Input: "你好呀", ExpectedIntents: []string{"chitchat"}},
	// {Input: "你是谁", ExpectedIntents: []string{"chitchat"}},
	// {Input: "今天天气怎么样", ExpectedIntents: []string{"chitchat"}},
	// {Input: "谢谢你", ExpectedIntents: []string{"chitchat"}},
	// {Input: "你能做什么", ExpectedIntents: []string{"chitchat"}},
	// {Input: "早上好", ExpectedIntents: []string{"chitchat"}},
	// {Input: "你今天心情怎么样", ExpectedIntents: []string{"chitchat"}},
	// {Input: "讲个笑话", ExpectedIntents: []string{"chitchat"}},
	// {Input: "你会不会唱歌", ExpectedIntents: []string{"chitchat"}},
	// {Input: "我有点无聊", ExpectedIntents: []string{"chitchat"}},

	// // 边界 / 单意图
	// {Input: "耐克和阿迪哪个好", ExpectedIntents: []string{"product_search"}},
	// {Input: "上次买的鞋子质量很差想换货", ExpectedIntents: []string{"customer_service"}},
	// {Input: "有没有比昨天买的那双更便宜的", ExpectedIntents: []string{"product_search"}},
	// {Input: "我买错尺码了", ExpectedIntents: []string{"customer_service"}},
	// {Input: "你们家耳机音质怎么样", ExpectedIntents: []string{"query_product_performance"}},
	// {Input: "我昨天买的耳机能不能补差价", ExpectedIntents: []string{"customer_service"}},

	// 多意图混合 - 后端将独立处理两个流程
	{Input: "先推荐一款手机，再告诉我退货规则", ExpectedIntents: []string{"customer_service", "product_search"}},
	{Input: "我的包裹丢了顺便推荐个行李箱", ExpectedIntents: []string{"customer_service", "product_search"}},
	{Input: "我想买个平板电脑，顺便问一下售后服务", ExpectedIntents: []string{"customer_service", "product_search"}},
	{Input: "我想买个新手机，之前买的那个有质量问题", ExpectedIntents: []string{"customer_service", "product_search"}},
	{Input: "我想买个新手机，之前买的那个有质量问题，你们能不能帮我换一个", ExpectedIntents: []string{"customer_service", "product_search"}},

	{Input: "我想买个微波炉，要是坏了你们上门维修吗", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "帮 me 推荐个运动相机，顺便问下昨天那个订单怎么退款", ExpectedIntents: []string{"product_search", "customer_service"}},

	{Input: "你们店里有新款帐篷吗？发货到上海要多久", ExpectedIntents: []string{"product_search", "customer_service"}},

	{Input: "我想买个平板，顺便问下你们保修几年", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "刚才那个订单我选错尺寸了，能不能改，还有帮我推荐个搭配的裤子", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "之前的耳机断了质量太差了，我要退货，顺便看下有没有更结实的推荐", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "能不能帮我催下物流，顺便再找一个同款的链接给我也想给朋友买一个", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "有没有显白的口红？之前买的那支颜色发错了我要投诉", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "帮我找找有没有白色的衬衫，顺便问下如果尺码不合适包退换吗", ExpectedIntents: []string{"product_search", "customer_service"}},

	{Input: "推荐几本书，顺便问下满100包邮吗", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "这个项链会掉色吗，想买个不掉色的送人，有推荐吗", ExpectedIntents: []string{"query_product_performance", "product_search"}},
	{Input: "我想买个防晒霜，另外之前那个包裹没收到显示签收了怎么办", ExpectedIntents: []string{"product_search", "customer_service"}},
	{Input: "给我介绍下这款相机的光圈参数", ExpectedIntents: []string{"query_product_performance"}},

	{Input: "有没有那种防滑的拖鞋，顺便问下你们发货地在哪里", ExpectedIntents: []string{"product_search", "customer_service"}},
}

// ── 意图解析 ──────────────────────────────────────────────────

type IntentResult struct {
	Intents []string `json:"intents"`
}

func (r *IntentResult) UnmarshalJSON(data []byte) error {
	var raw struct {
		Intents json.RawMessage `json:"intents"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch {
	case len(raw.Intents) == 0:
		r.Intents = nil
		return nil
	case raw.Intents[0] == '[':
		return json.Unmarshal(raw.Intents, &r.Intents)
	case raw.Intents[0] == '"':
		var intent string
		if err := json.Unmarshal(raw.Intents, &intent); err != nil {
			return err
		}
		r.Intents = []string{intent}
		return nil
	default:
		return fmt.Errorf("不支持的 intents 格式: %s", string(raw.Intents))
	}
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}

func detectIntent(ctx context.Context, client *IntentClient, message string) (*IntentResult, string, time.Duration, error) {
	start := time.Now()

	raw, err := client.Complete(ctx, intentPrompt+message)
	elapsed := time.Since(start)
	if err != nil {
		return nil, "", elapsed, err
	}

	raw = extractJSON(raw)

	var result IntentResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, raw, elapsed, fmt.Errorf("JSON解析失败: %w", err)
	}

	if len(result.Intents) == 0 {
		return nil, raw, elapsed, fmt.Errorf("intents 为空，可能是解析错误")
	}

	return &result, raw, elapsed, nil
}

// 检查实际意图是否包含所有期望意图
func getDetectedIntents(result *IntentResult) []string {
	return result.Intents
}

func containsIntent(intents []string, target string) bool {
	for _, item := range intents {
		if item == target {
			return true
		}
	}
	return false
}

func containsAllIntents(actual []string, expected []string) bool {
	for _, need := range expected {
		if !containsIntent(actual, need) {
			return false
		}
	}
	return true
}

// ── 主测试逻辑 ────────────────────────────────────────────────

func main() {
	ctx := context.Background()

	client := NewIntentClient("http://localhost:11434", "qwen3.5:0.8b")

	fmt.Println("开始测试多意图独立评分...")
	fmt.Println(strings.Repeat("─", 100))

	var (
		total     = len(testCases)
		correct   = 0
		jsonFail  = 0
		totalTime time.Duration
	)

	for _, tc := range testCases {
		result, raw, elapsed, err := detectIntent(ctx, client, tc.Input)
		totalTime += elapsed

		if err != nil {
			jsonFail++
			fmt.Printf("❌  [JSON错误] %-40s → %v\n", tc.Input, err)
			continue
		}

		// 获取检测到的意图（score > 0.5）
		detectedIntents := getDetectedIntents(result)

		// 检查是否包含所有期望的意图
		allIntentsHit := containsAllIntents(detectedIntents, tc.ExpectedIntents)

		// 额外检查：不应该有期望外的高分意图
		unexpectedIntentFound := false
		for _, detected := range detectedIntents {
			if !containsIntent(tc.ExpectedIntents, detected) {
				unexpectedIntentFound = true
				break
			}
		}

		isCorrect := allIntentsHit && !unexpectedIntentFound
		if isCorrect {
			correct++
		}

		status := "✅"
		if !isCorrect {
			status = "❌"
		}

		// 输出格式：输入 → 检测到的意图 期望的意图
		fmt.Printf("%s  %-40s → 检测:%v 期望:%v\n",
			status,
			tc.Input,
			detectedIntents,
			tc.ExpectedIntents,
		)

		if !isCorrect {
			fmt.Printf("    原始输出: %s\n", raw)
		}
	}

	// ── 汇总报告 ──────────────────────────────────────────────
	fmt.Println(strings.Repeat("─", 100))
	accuracy := float64(correct) / float64(total) * 100
	avgTime := totalTime / time.Duration(total)

	fmt.Printf("\n📊 测试报告\n")
	fmt.Printf("   总用例数:     %d\n", total)
	fmt.Printf("   正确数:       %d\n", correct)
	fmt.Printf("   JSON解析失败: %d\n", jsonFail)
	fmt.Printf("   准确率:       %.1f%%\n", accuracy)
	fmt.Printf("   平均耗时:     %dms\n", avgTime.Milliseconds())
	fmt.Println()

	switch {
	case accuracy >= 95:
		fmt.Println("🎉 结论：完全满足需求，可以直接上线")
	case accuracy >= 85:
		fmt.Println("⚠️  结论：基本满足需求，建议优化 Prompt 后再上线")
		fmt.Println("   建议：在 Prompt 里增加更多 few-shot 示例")
	case accuracy >= 70:
		fmt.Println("⚠️  结论：准确率偏低，需要改进")
		fmt.Println("   建议：检查失败用例，针对性补充示例")
	default:
		fmt.Println("❌ 结论：不满足需求，建议换用更大的模型")
		fmt.Println("   建议：尝试 qwen2.5:14b 或调用云端 API")
	}
}

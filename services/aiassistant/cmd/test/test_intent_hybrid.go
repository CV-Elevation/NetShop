// cmd/test/test_intent_hybrid.go
// 用法：go run test_intent_hybrid.go
// 说明：朴素贝叶斯先判，低置信度时调用 LLM 兜底重判

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type TestCase struct {
	Input           string
	ExpectedIntents []string
}

var testCases = []TestCase{
	{Input: "有没有防水的登山鞋", ExpectedIntents: []string{"product_search"}},
	{Input: "推荐一款适合跑步的鞋子", ExpectedIntents: []string{"product_search"}},
	{Input: "我想买一双耐克运动鞋", ExpectedIntents: []string{"product_search"}},
	{Input: "有没有红色的连衣裙", ExpectedIntents: []string{"product_search"}},
	{Input: "200块以内的蓝牙耳机有哪些", ExpectedIntents: []string{"product_search"}},
	{Input: "帮我找找保暖内衣", ExpectedIntents: []string{"product_search"}},
	{Input: "有没有儿童书包推荐", ExpectedIntents: []string{"product_search"}},
	{Input: "想看轻薄一点的笔记本电脑", ExpectedIntents: []string{"product_search"}},
	{Input: "给我推荐适合露营用的手电筒", ExpectedIntents: []string{"product_search"}},
	{Input: "有没有支持降噪的头戴式耳机", ExpectedIntents: []string{"product_search"}},
	{Input: "比较一下你们家两款空气炸锅", ExpectedIntents: []string{"product_search"}},
	{Input: "我预算500以内买个机械键盘", ExpectedIntents: []string{"product_search"}},
	{Input: "我的订单还没发货怎么办", ExpectedIntents: []string{"customer_service"}},
	{Input: "退货流程是什么", ExpectedIntents: []string{"customer_service"}},
	{Input: "我想申请退款", ExpectedIntents: []string{"customer_service"}},
	{Input: "快递一直显示在途怎么回事", ExpectedIntents: []string{"customer_service"}},
	{Input: "收到的商品有破损", ExpectedIntents: []string{"customer_service"}},
	{Input: "发票怎么开", ExpectedIntents: []string{"customer_service"}},
	{Input: "我的账号被封了", ExpectedIntents: []string{"customer_service"}},
	{Input: "优惠券用不了", ExpectedIntents: []string{"customer_service"}},
	{Input: "我支付成功了但订单还是未支付", ExpectedIntents: []string{"customer_service"}},
	{Input: "可以帮我催一下快递吗", ExpectedIntents: []string{"customer_service"}},
	{Input: "这单我想改收货地址", ExpectedIntents: []string{"customer_service"}},
	{Input: "我买到假货了怎么处理", ExpectedIntents: []string{"customer_service"}},
	{Input: "售后电话是多少", ExpectedIntents: []string{"customer_service"}},
	{Input: "你好呀", ExpectedIntents: []string{"chitchat"}},
	{Input: "你是谁", ExpectedIntents: []string{"chitchat"}},
	{Input: "今天天气怎么样", ExpectedIntents: []string{"chitchat"}},
	{Input: "谢谢你", ExpectedIntents: []string{"chitchat"}},
	{Input: "你能做什么", ExpectedIntents: []string{"chitchat"}},
	{Input: "早上好", ExpectedIntents: []string{"chitchat"}},
	{Input: "你今天心情怎么样", ExpectedIntents: []string{"chitchat"}},
	{Input: "讲个笑话", ExpectedIntents: []string{"chitchat"}},
	{Input: "你会不会唱歌", ExpectedIntents: []string{"chitchat"}},
	{Input: "我有点无聊", ExpectedIntents: []string{"chitchat"}},
	{Input: "耐克和阿迪哪个好", ExpectedIntents: []string{"product_search"}},
	{Input: "上次买的鞋子质量很差想换货", ExpectedIntents: []string{"customer_service"}},
	{Input: "有没有比昨天买的那双更便宜的", ExpectedIntents: []string{"product_search"}},
	{Input: "我买错尺码了", ExpectedIntents: []string{"customer_service"}},
	{Input: "你们家耳机音质怎么样", ExpectedIntents: []string{"query_product_performance"}},
	{Input: "我昨天买的耳机能不能补差价", ExpectedIntents: []string{"customer_service"}},
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

type IntentResult struct {
	Intents    []string           `json:"intents"`
	Confidence map[string]float64 `json:"confidence,omitempty"`
	Source     string             `json:"source,omitempty"`
}

type NaiveBayesClassifier struct {
	intents      []string
	vocab        map[string]struct{}
	priorPos     map[string]float64
	tokenProbPos map[string]map[string]float64
	tokenProbNeg map[string]map[string]float64
	threshold    float64
	alpha        float64
}

func NewNaiveBayesClassifier(train []TestCase) *NaiveBayesClassifier {
	c := &NaiveBayesClassifier{
		intents:      []string{"product_search", "customer_service", "query_product_performance", "chitchat"},
		vocab:        make(map[string]struct{}),
		priorPos:     make(map[string]float64),
		tokenProbPos: make(map[string]map[string]float64),
		tokenProbNeg: make(map[string]map[string]float64),
		threshold:    getEnvFloat64("NB_INTENT_THRESHOLD", 0.80),
		alpha:        1.0,
	}

	for _, tc := range train {
		for _, tok := range tokenize(tc.Input) {
			c.vocab[tok] = struct{}{}
		}
	}
	vSize := float64(len(c.vocab))
	n := float64(len(train))

	for _, intent := range c.intents {
		posDocs := 0.0
		negDocs := 0.0
		posCounts := map[string]float64{}
		negCounts := map[string]float64{}
		totalPosTokens := 0.0
		totalNegTokens := 0.0

		for _, tc := range train {
			toks := tokenize(tc.Input)
			if containsIntent(tc.ExpectedIntents, intent) {
				posDocs++
				for _, t := range toks {
					posCounts[t]++
					totalPosTokens++
				}
			} else {
				negDocs++
				for _, t := range toks {
					negCounts[t]++
					totalNegTokens++
				}
			}
		}

		c.priorPos[intent] = (posDocs + c.alpha) / (n + 2*c.alpha)
		c.tokenProbPos[intent] = map[string]float64{}
		c.tokenProbNeg[intent] = map[string]float64{}

		for tok := range c.vocab {
			c.tokenProbPos[intent][tok] = (posCounts[tok] + c.alpha) / (totalPosTokens + c.alpha*vSize)
			c.tokenProbNeg[intent][tok] = (negCounts[tok] + c.alpha) / (totalNegTokens + c.alpha*vSize)
		}
	}

	return c
}

func (c *NaiveBayesClassifier) Predict(message string) IntentResult {
	toks := tokenize(message)
	detected := make([]string, 0, 2)
	confidence := make(map[string]float64, len(c.intents))

	for _, intent := range c.intents {
		p := c.scoreIntent(intent, toks)
		confidence[intent] = p
		if p >= c.threshold {
			detected = append(detected, intent)
		}
	}

	if len(detected) == 0 {
		detected = []string{"chitchat"}
	}
	sort.Strings(detected)
	return IntentResult{Intents: detected, Confidence: confidence, Source: "nb"}
}

func (c *NaiveBayesClassifier) scoreIntent(intent string, toks []string) float64 {
	pPos := c.priorPos[intent]
	pNeg := 1.0 - pPos

	logPos := math.Log(pPos)
	logNeg := math.Log(pNeg)

	for _, t := range toks {
		if _, ok := c.vocab[t]; !ok {
			continue
		}
		logPos += math.Log(c.tokenProbPos[intent][t])
		logNeg += math.Log(c.tokenProbNeg[intent][t])
	}

	return 1.0 / (1.0 + math.Exp(logNeg-logPos))
}

func tokenize(text string) []string {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return nil
	}

	asciiTokens := make([]string, 0)
	var buf []rune
	flush := func() {
		if len(buf) > 0 {
			asciiTokens = append(asciiTokens, string(buf))
			buf = nil
		}
	}

	runes := []rune(t)
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf = append(buf, r)
		} else {
			flush()
		}
	}
	flush()

	bigrams := make([]string, 0)
	for i := 0; i < len(runes)-1; i++ {
		r1, r2 := runes[i], runes[i+1]
		if unicode.IsSpace(r1) || unicode.IsSpace(r2) {
			continue
		}
		if strings.ContainsRune("，。！？；：,.!?()[]{}\"' ", r1) || strings.ContainsRune("，。！？；：,.!?()[]{}\"' ", r2) {
			continue
		}
		bigrams = append(bigrams, string([]rune{r1, r2}))
	}

	out := append(asciiTokens, bigrams...)
	if len(out) == 0 {
		return []string{t}
	}
	return out
}

type IntentClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewIntentClient(baseURL, model string, timeout time.Duration) *IntentClient {
	return &IntentClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
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

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}

func llmDetectIntent(ctx context.Context, client *IntentClient, message string) (IntentResult, string, time.Duration, error) {
	start := time.Now()
	raw, err := client.Complete(ctx, intentPrompt+message)
	elapsed := time.Since(start)
	if err != nil {
		return IntentResult{}, "", elapsed, err
	}

	raw = extractJSON(raw)

	var parsed struct {
		Intents json.RawMessage `json:"intents"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return IntentResult{}, raw, elapsed, fmt.Errorf("JSON解析失败: %w", err)
	}

	var intents []string
	switch {
	case len(parsed.Intents) == 0:
		intents = nil
	case parsed.Intents[0] == '[':
		if err := json.Unmarshal(parsed.Intents, &intents); err != nil {
			return IntentResult{}, raw, elapsed, err
		}
	case parsed.Intents[0] == '"':
		var single string
		if err := json.Unmarshal(parsed.Intents, &single); err != nil {
			return IntentResult{}, raw, elapsed, err
		}
		intents = []string{single}
	default:
		return IntentResult{}, raw, elapsed, fmt.Errorf("不支持的 intents 格式: %s", string(parsed.Intents))
	}

	sort.Strings(intents)
	return IntentResult{Intents: intents, Source: "llm"}, raw, elapsed, nil
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

func isExactIntentMatch(actual []string, expected []string) bool {
	if !containsAllIntents(actual, expected) {
		return false
	}
	for _, got := range actual {
		if !containsIntent(expected, got) {
			return false
		}
	}
	return true
}

func getEnvFloat64(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvInt64(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func splitRandom(cases []TestCase, trainRatio float64, seed int64) ([]TestCase, []TestCase) {
	if len(cases) <= 1 {
		return cases, nil
	}
	if trainRatio <= 0 {
		trainRatio = 0.7
	}
	if trainRatio >= 1 {
		trainRatio = 0.9
	}

	idx := make([]int, len(cases))
	for i := range idx {
		idx[i] = i
	}

	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(idx), func(i, j int) {
		idx[i], idx[j] = idx[j], idx[i]
	})

	trainSize := int(math.Round(float64(len(cases)) * trainRatio))
	if trainSize < 1 {
		trainSize = 1
	}
	if trainSize >= len(cases) {
		trainSize = len(cases) - 1
	}

	train := make([]TestCase, 0, trainSize)
	test := make([]TestCase, 0, len(cases)-trainSize)
	for i, original := range idx {
		if i < trainSize {
			train = append(train, cases[original])
		} else {
			test = append(test, cases[original])
		}
	}
	return train, test
}

func confidenceOfPrediction(res IntentResult) float64 {
	if len(res.Confidence) == 0 {
		return 0
	}

	if len(res.Intents) == 0 {
		maxP := 0.0
		for _, p := range res.Confidence {
			if p > maxP {
				maxP = p
			}
		}
		return maxP
	}

	minP := 1.0
	for _, in := range res.Intents {
		p, ok := res.Confidence[in]
		if !ok {
			return 0
		}
		if p < minP {
			minP = p
		}
	}
	return minP
}

func formatConfidence(scores map[string]float64, order []string) string {
	parts := make([]string, 0, len(order))
	for _, intent := range order {
		parts = append(parts, fmt.Sprintf("%s=%.2f", intent, scores[intent]))
	}
	return strings.Join(parts, ", ")
}

func main() {
	ctx := context.Background()

	trainRatio := getEnvFloat64("INTENT_TRAIN_RATIO", 0.7)
	seed := getEnvInt64("INTENT_EVAL_SEED", 42)
	fallbackThreshold := getEnvFloat64("HYBRID_FALLBACK_CONFIDENCE", 0.75)
	llmBaseURL := strings.TrimSpace(os.Getenv("HYBRID_LLM_BASE_URL"))
	if llmBaseURL == "" {
		llmBaseURL = "http://localhost:11434"
	}
	llmModel := strings.TrimSpace(os.Getenv("HYBRID_LLM_MODEL"))
	if llmModel == "" {
		llmModel = "qwen3.5:0.8b"
	}
	llmTimeoutMS := getEnvInt64("HYBRID_LLM_TIMEOUT_MS", 30000)

	trainSet, testSet := splitRandom(testCases, trainRatio, seed)
	if len(testSet) == 0 {
		fmt.Println("❌ 测试集为空，请调小 INTENT_TRAIN_RATIO")
		return
	}

	nb := NewNaiveBayesClassifier(trainSet)
	llm := NewIntentClient(llmBaseURL, llmModel, time.Duration(llmTimeoutMS)*time.Millisecond)

	fmt.Println("开始测试混合意图识别（NB + LLM 兜底）...")
	fmt.Printf("数据划分: 训练集=%d, 测试集=%d, 训练比例=%.2f, 随机种子=%d\n", len(trainSet), len(testSet), trainRatio, seed)
	fmt.Printf("兜底阈值: %.2f, LLM: %s (%s)\n", fallbackThreshold, llmModel, llmBaseURL)
	fmt.Println(strings.Repeat("─", 120))

	var (
		total             = len(testSet)
		nbOnlyCorrect     = 0
		hybridCorrect     = 0
		fallbackTriggered = 0
		fallbackLLMErrors = 0
		totalNBTime       time.Duration
		totalFallbackTime time.Duration
	)

	for _, tc := range testSet {
		nbStart := time.Now()
		nbRes := nb.Predict(tc.Input)
		nbElapsed := time.Since(nbStart)
		totalNBTime += nbElapsed

		nbConf := confidenceOfPrediction(nbRes)
		nbCorrect := isExactIntentMatch(nbRes.Intents, tc.ExpectedIntents)
		if nbCorrect {
			nbOnlyCorrect++
		}

		finalRes := nbRes
		usedFallback := nbConf < fallbackThreshold
		llmRaw := ""

		if usedFallback {
			fallbackTriggered++
			llmRes, raw, llmElapsed, err := llmDetectIntent(ctx, llm, tc.Input)
			totalFallbackTime += llmElapsed
			llmRaw = raw
			if err == nil && len(llmRes.Intents) > 0 {
				finalRes = llmRes
			} else if err != nil {
				fallbackLLMErrors++
			}
		}

		finalCorrect := isExactIntentMatch(finalRes.Intents, tc.ExpectedIntents)
		if finalCorrect {
			hybridCorrect++
		}

		status := "✅"
		if !finalCorrect {
			status = "❌"
		}

		source := "NB"
		if finalRes.Source == "llm" {
			source = "LLM"
		}

		fmt.Printf("%s  %-34s → 最终:%v 期望:%v  来源:%s\n", status, tc.Input, finalRes.Intents, tc.ExpectedIntents, source)
		fmt.Printf("    NB置信度(min-selected): %.2f | NB各意图: %s\n", nbConf, formatConfidence(nbRes.Confidence, nb.intents))
		if usedFallback {
			fmt.Printf("    触发兜底: 是")
			if llmRaw != "" {
				fmt.Printf(" | LLM原始输出: %s", llmRaw)
			}
			fmt.Println()
		} else {
			fmt.Println("    触发兜底: 否")
		}
	}

	fmt.Println(strings.Repeat("─", 120))
	nbAcc := float64(nbOnlyCorrect) / float64(total) * 100
	hybridAcc := float64(hybridCorrect) / float64(total) * 100
	avgNB := totalNBTime / time.Duration(total)
	avgFallback := time.Duration(0)
	if fallbackTriggered > 0 {
		avgFallback = totalFallbackTime / time.Duration(fallbackTriggered)
	}

	fmt.Printf("\n📊 混合评测报告\n")
	fmt.Printf("   总用例数:             %d\n", total)
	fmt.Printf("   NB单独正确数:         %d\n", nbOnlyCorrect)
	fmt.Printf("   混合策略正确数:       %d\n", hybridCorrect)
	fmt.Printf("   NB单独准确率:         %.1f%%\n", nbAcc)
	fmt.Printf("   混合策略准确率:       %.1f%%\n", hybridAcc)
	fmt.Printf("   兜底触发次数:         %d (%.1f%%)\n", fallbackTriggered, float64(fallbackTriggered)/float64(total)*100)
	fmt.Printf("   LLM兜底失败次数:      %d\n", fallbackLLMErrors)
	fmt.Printf("   NB平均耗时:           %dms\n", avgNB.Milliseconds())
	fmt.Printf("   兜底LLM平均耗时:      %dms\n", avgFallback.Milliseconds())
	fmt.Println()

	switch {
	case hybridAcc >= 95:
		fmt.Println("🎉 结论：混合策略效果优秀，可用于上线前压测")
	case hybridAcc >= 85:
		fmt.Println("⚠️  结论：混合策略可用，建议继续调低置信度阈值并补充样本")
	case hybridAcc >= 70:
		fmt.Println("⚠️  结论：仍需优化，建议增加训练样本并调优兜底阈值")
	default:
		fmt.Println("❌ 结论：效果不足，建议进一步改造特征工程或提升 LLM 模型")
	}
}

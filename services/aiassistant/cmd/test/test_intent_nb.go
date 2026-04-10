// cmd/test/test_intent_nb.go
// 用法：go run test_intent_nb.go
// 说明：沿用 test_intent.go 的测试样例和报告结构，仅将识别逻辑替换为朴素贝叶斯

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
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
	// product_search 单意图
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

	// customer_service 单意图
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

	// chitchat 单意图
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

	// 边界 / 单意图
	{Input: "耐克和阿迪哪个好", ExpectedIntents: []string{"product_search"}},
	{Input: "上次买的鞋子质量很差想换货", ExpectedIntents: []string{"customer_service"}},
	{Input: "有没有比昨天买的那双更便宜的", ExpectedIntents: []string{"product_search"}},
	{Input: "我买错尺码了", ExpectedIntents: []string{"customer_service"}},
	{Input: "你们家耳机音质怎么样", ExpectedIntents: []string{"query_product_performance"}},
	{Input: "我昨天买的耳机能不能补差价", ExpectedIntents: []string{"customer_service"}},

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

type IntentResult struct {
	Intents    []string           `json:"intents"`
	Confidence map[string]float64 `json:"confidence"`
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
		threshold:    0.80,
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
	return IntentResult{Intents: detected, Confidence: confidence}
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

	// logit -> probability
	return 1.0 / (1.0 + math.Exp(logNeg-logPos))
}

func tokenize(text string) []string {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return nil
	}

	// 1) 连续英文/数字词
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

	// 2) 中文/混合二元组
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

func detectIntent(ctx context.Context, classifier *NaiveBayesClassifier, message string) (*IntentResult, string, time.Duration, error) {
	_ = ctx
	start := time.Now()
	result := classifier.Predict(message)
	elapsed := time.Since(start)
	rawBytes, err := json.Marshal(result)
	if err != nil {
		return nil, "", elapsed, err
	}
	return &result, string(rawBytes), elapsed, nil
}

func formatConfidence(scores map[string]float64, order []string) string {
	parts := make([]string, 0, len(order))
	for _, intent := range order {
		parts = append(parts, fmt.Sprintf("%s=%.2f", intent, scores[intent]))
	}
	return strings.Join(parts, ", ")
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

func main() {
	ctx := context.Background()

	trainRatio := getEnvFloat64("INTENT_TRAIN_RATIO", 0.7)
	seed := getEnvInt64("INTENT_EVAL_SEED", 42)
	trainSet, testSet := splitRandom(testCases, trainRatio, seed)

	if len(testSet) == 0 {
		fmt.Println("❌ 测试集为空，请调小 INTENT_TRAIN_RATIO")
		return
	}

	classifier := NewNaiveBayesClassifier(trainSet)

	fmt.Println("开始测试朴素贝叶斯意图识别（随机切分）...")
	fmt.Printf("数据划分: 训练集=%d, 测试集=%d, 训练比例=%.2f, 随机种子=%d\n", len(trainSet), len(testSet), trainRatio, seed)
	fmt.Println(strings.Repeat("─", 100))

	var (
		total     = len(testSet)
		correct   = 0
		jsonFail  = 0
		totalTime time.Duration
	)

	for _, tc := range testSet {
		result, raw, elapsed, err := detectIntent(ctx, classifier, tc.Input)
		totalTime += elapsed

		if err != nil {
			jsonFail++
			fmt.Printf("❌  [JSON错误] %-40s → %v\n", tc.Input, err)
			continue
		}

		detectedIntents := result.Intents
		allIntentsHit := containsAllIntents(detectedIntents, tc.ExpectedIntents)

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

		fmt.Printf("%s  %-40s → 检测:%v 期望:%v\n", status, tc.Input, detectedIntents, tc.ExpectedIntents)
		fmt.Printf("    置信度: %s\n", formatConfidence(result.Confidence, classifier.intents))
		if !isCorrect {
			fmt.Printf("    原始输出: %s\n", raw)
		}
	}

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
		fmt.Println("⚠️  结论：基本满足需求，建议补充训练样本后再上线")
	case accuracy >= 70:
		fmt.Println("⚠️  结论：准确率偏低，需要改进")
		fmt.Println("   建议：补充训练样本，或切换到混合规则+贝叶斯")
	default:
		fmt.Println("❌ 结论：不满足需求，建议继续使用 LLM 识别或混合方案")
	}
}

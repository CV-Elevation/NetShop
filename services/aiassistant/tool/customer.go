package tool

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type CustomerTool struct{}

func NewCustomerTool() *CustomerTool {
	return &CustomerTool{}
}

type CustomerToolParam struct {
	Question string `json:"question"`
}
type faqItem struct {
	Keywords []string
	Answer   string
}

var customerFAQ = []faqItem{
	{
		Keywords: []string{"退货", "退款", "退钱", "return", "refund"},
		Answer:   "支持7天无理由退货（部分特殊商品除外）。如商品无使用痕迹，请在订单详情发起退货申请。",
	},
	{
		Keywords: []string{"运费", "邮费", "包邮", "shipping"},
		Answer:   "普通订单满99元包邮；未满99元收取8元运费，偏远地区运费以结算页为准。",
	},
	{
		Keywords: []string{"发货", "多久", "何时", "物流", "快递", "delivery"},
		Answer:   "工作日16:00前付款的订单一般当天发货，之后下单通常次日发货。",
	},
	{
		Keywords: []string{"发票", "invoice"},
		Answer:   "下单时可选择电子发票，开票后会发送至您的邮箱，并可在订单详情中下载。",
	},
	{
		Keywords: []string{"支付", "付款", "pay", "支付宝", "微信", "银行卡"},
		Answer:   "目前支持支付宝、微信支付和主流银行卡支付，具体以收银台展示为准。",
	},
	{
		Keywords: []string{"改地址", "修改地址", "收货地址", "address"},
		Answer:   "订单未发货前可在订单详情尝试修改收货地址；若已发货，请联系物流或客服协助。",
	},
}

func (t *CustomerTool) ToolName() AgentTool {
	return AgentToolCustomer
}

func (t *CustomerTool) Info() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
		Name:        string(AgentToolCustomer),
		Description: openai.String("Act as an Intelligent Customer Service,deal with user's question about shopping policy"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type":        "string",
					"description": "customer question about shopping policy",
				},
			},
			"required": []string{"question"},
		},
	})
}

func (t *CustomerTool) Execute(ctx context.Context, argumentsInJSON string) (string, error) {
	p := CustomerToolParam{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", err
	}

	question := strings.ToLower(strings.TrimSpace(p.Question))
	if question == "" {
		return "请告诉我您想咨询的问题，例如：退货、运费、发货时间、发票或支付方式。", nil
	}

	bestScore := 0
	bestAnswer := "暂时没有完全匹配到您的问题。您可以换个说法，或咨询：退货退款、运费、发货时间、发票、支付方式、地址修改。"

	for _, item := range customerFAQ {
		score := 0
		for _, kw := range item.Keywords {
			k := strings.ToLower(kw)
			if strings.Contains(question, k) {
				score += 2
			}
			if strings.Contains(k, question) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestAnswer = item.Answer
		}
	}

	return bestAnswer, nil
}

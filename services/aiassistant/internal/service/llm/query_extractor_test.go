package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExtractQueryJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain json",
			in:   `{"keyword":"蓝牙耳机","max_price":30000,"min_price":0,"category":""}`,
			want: `{"keyword":"蓝牙耳机","max_price":30000,"min_price":0,"category":""}`,
		},
		{
			name: "json with prefix suffix",
			in:   "结果如下：\n{\"keyword\":\"洗碗机\",\"max_price\":0,\"min_price\":0,\"category\":\"\"}\n谢谢",
			want: `{"keyword":"洗碗机","max_price":0,"min_price":0,"category":""}`,
		},
		{
			name: "no json",
			in:   "没有 json",
			want: "没有 json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractQueryJSON(tc.in)
			if got != tc.want {
				t.Fatalf("extractQueryJSON() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeQueryEndpoint(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "http://localhost:11434/api/chat"},
		{name: "base url", in: "http://localhost:11434", want: "http://localhost:11434/api/chat"},
		{name: "generate url", in: "http://localhost:11434/api/generate", want: "http://localhost:11434/api/chat"},
		{name: "chat url", in: "http://localhost:11434/api/chat", want: "http://localhost:11434/api/chat"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeQueryEndpoint(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeQueryEndpoint() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProductQueryExtractor_Extract_Integration(t *testing.T) {
	// if os.Getenv("PRODUCT_QUERY_LLM_INTEGRATION") != "true" {
	// 	t.Skip("set PRODUCT_QUERY_LLM_INTEGRATION=true to run against local LLM")
	// }

	extractor := NewProductQueryExtractorFromEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	testCases := []struct {
		name  string
		input string
		want  ProductSearchQuery
	}{
		{
			name:  "price upper bound",
			input: "300元以内的蓝牙耳机",
			want:  ProductSearchQuery{Keyword: "蓝牙耳机", MaxPrice: 30000, MinPrice: 0, Category: ""},
		},
		{
			name:  "price lower bound",
			input: "1000元以上的平板电脑",
			want:  ProductSearchQuery{Keyword: "平板电脑", MaxPrice: 0, MinPrice: 100000, Category: ""},
		},
		{
			name:  "category and keyword",
			input: "角色模型，想看Holo的手办",
			want:  ProductSearchQuery{Keyword: "Holo手办", MaxPrice: 0, MinPrice: 0, Category: "角色模型"},
		},
		{
			name:  "both bounds",
			input: "200到500元的机械键盘",
			want:  ProductSearchQuery{Keyword: "机械键盘", MaxPrice: 50000, MinPrice: 20000, Category: ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := extractor.Extract(ctx, tc.input)
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}

			if result.Keyword == "" {
				t.Fatalf("Extract() keyword is empty, want %q", tc.want.Keyword)
			}

			if tc.want.Keyword != "" && !stringsContainsAll(result.Keyword, tc.want.Keyword) {
				t.Fatalf("Extract() keyword = %q, want to contain %q", result.Keyword, tc.want.Keyword)
			}

			if tc.want.Category != "" && !stringsContainsAll(result.Category, tc.want.Category) {
				t.Fatalf("Extract() category = %q, want to contain %q", result.Category, tc.want.Category)
			}

			if tc.want.MaxPrice > 0 && result.MaxPrice != tc.want.MaxPrice {
				t.Fatalf("Extract() max_price = %d, want %d", result.MaxPrice, tc.want.MaxPrice)
			}

			if tc.want.MinPrice > 0 && result.MinPrice != tc.want.MinPrice {
				t.Fatalf("Extract() min_price = %d, want %d", result.MinPrice, tc.want.MinPrice)
			}
		})
	}
}

func stringsContainsAll(got, want string) bool {
	got = normalizeForCompare(got)
	want = normalizeForCompare(want)
	return strings.Contains(got, want)
}

func normalizeForCompare(s string) string {
	return removeSpaces(strings.TrimSpace(s))
}

func removeSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, s)
}

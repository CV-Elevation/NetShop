// cmd/import_customer_service_doc/main.go
// 用法：go run ./cmd/import_customer_service_doc

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"netshop/services/aiassistant/internal/repository"
	"netshop/services/aiassistant/internal/service/llm"
)

const customerServiceDocSource = "customer-service-guide-v1"

type docChunk struct {
	Title   string
	Summary string
	Text    string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	docPath := resolveDocPath()
	content, err := os.ReadFile(docPath)
	if err != nil {
		panic(fmt.Errorf("read doc failed: %w", err))
	}

	chunks := splitMarkdownChunks(string(content))
	if len(chunks) == 0 {
		panic("no chunks extracted from document")
	}

	embedder := llm.NewDoubaoClientFromEnv()
	dsn := os.Getenv("KNOWLEDGE_DB_DSN")
	if dsn == "" {
		dsn = "postgres://netshop:secret@localhost:5432/netshop?sslmode=disable"
	}

	repo, err := repository.NewKnowledgeBaseRepository(ctx, dsn, embedder)
	if err != nil {
		panic(fmt.Errorf("open knowledge repo failed: %w", err))
	}
	defer repo.Close()

	if err := repo.DeleteChunksBySource(ctx, customerServiceDocSource); err != nil {
		panic(fmt.Errorf("delete old doc chunks failed: %w", err))
	}

	for _, chunk := range chunks {
		embedding, err := embedder.Embed(ctx, chunk.Text)
		if err != nil {
			panic(fmt.Errorf("embed chunk failed: %w", err))
		}
		if err := repo.InsertChunk(ctx, chunk.Title, chunk.Summary, chunk.Text, customerServiceDocSource, embedding); err != nil {
			panic(fmt.Errorf("insert chunk failed: %w", err))
		}
	}

	fmt.Printf("imported %d chunks from %s\n", len(chunks), docPath)
}

func resolveDocPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot resolve current file path")
	}
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	return filepath.Join(baseDir, "docs", "customer_service_guide.md")
}

func splitMarkdownChunks(content string) []docChunk {
	sections := strings.Split(content, "\n## ")
	chunks := make([]docChunk, 0, len(sections))

	for i, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		title := "文档简介"
		body := section
		if i > 0 {
			lines := strings.SplitN(section, "\n", 2)
			title = strings.TrimSpace(lines[0])
			if len(lines) > 1 {
				body = strings.TrimSpace(lines[1])
			} else {
				body = ""
			}
		}

		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}

		chunks = append(chunks, docChunk{
			Title:   title,
			Summary: title,
			Text:    fmt.Sprintf("## %s\n\n%s", title, body),
		})
	}

	if len(chunks) == 0 {
		scanner := bufio.NewScanner(strings.NewReader(content))
		var lines []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			chunks = append(chunks, docChunk{
				Title:   "客服知识文档",
				Summary: "客服知识文档",
				Text:    strings.Join(lines, "\n"),
			})
		}
	}

	return chunks
}

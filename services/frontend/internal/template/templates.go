package template

import (
	"embed"
	"html/template"
)

//go:embed *.html
var htmlFS embed.FS

func Parse() (*template.Template, error) {
	return template.ParseFS(htmlFS, "*.html")
}

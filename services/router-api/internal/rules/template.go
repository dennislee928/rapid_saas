package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

type TemplateRenderer struct {
	funcs template.FuncMap
}

func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{funcs: template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"default": func(fallback, value any) any {
			if value == nil || value == "" {
				return fallback
			}
			return value
		},
		"now": func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
		"trunc": func(limit int, value any) string {
			text := fmt.Sprint(value)
			if len(text) <= limit {
				return text
			}
			return text[:limit]
		},
		"severityToColor": severityToColor,
		"ipInList": func(any, string) bool {
			return false
		},
	}}
}

func (r *TemplateRenderer) Render(kind, body string, data map[string]any, raw json.RawMessage) ([]byte, error) {
	switch kind {
	case "", "passthrough":
		return raw, nil
	case "template":
		tmpl, err := template.New("transform").Funcs(r.funcs).Parse(body)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "preset":
		return nil, fmt.Errorf("preset transform %q not implemented", body)
	default:
		return nil, fmt.Errorf("unsupported transform kind %q", kind)
	}
}

func EventData(headers map[string]string, raw json.RawMessage) map[string]any {
	data := map[string]any{"headers": headers, "raw": string(raw)}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err == nil {
		data["body"] = body
		for key, value := range body {
			data[key] = value
		}
	}
	return data
}

func severityToColor(value any) string {
	switch strings.ToLower(fmt.Sprint(value)) {
	case "critical":
		return "#d1242f"
	case "high":
		return "#fb8500"
	case "medium":
		return "#eac54f"
	case "low":
		return "#2da44e"
	default:
		return "#8c959f"
	}
}

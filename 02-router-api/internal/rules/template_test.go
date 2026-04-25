package rules

import (
	"encoding/json"
	"testing"
)

func TestTemplateRenderer(t *testing.T) {
	renderer := NewTemplateRenderer()
	out, err := renderer.Render("template", `{"text":"{{ .severity | upper }} {{ default "unknown" .user }}"}`, map[string]any{"severity": "high"}, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if string(out) != `{"text":"HIGH unknown"}` {
		t.Fatalf("unexpected output %s", out)
	}
}

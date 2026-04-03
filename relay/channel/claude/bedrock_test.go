package claude

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestSanitizeBedrockPromptCachingScope(t *testing.T) {
	payload := map[string]interface{}{
		"cache_control": map[string]interface{}{
			"type":  "ephemeral",
			"scope": "workspace",
		},
		"metadata": map[string]interface{}{
			"scope": "keep-me",
		},
		"system": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "system text",
			},
			map[string]interface{}{
				"type": "text",
				"text": "cached system text",
				"cache_control": map[string]interface{}{
					"type":  "ephemeral",
					"scope": "conversation",
				},
			},
		},
		"messages": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "hello",
						"cache_control": map[string]interface{}{
							"type":  "ephemeral",
							"scope": "message",
						},
					},
				},
			},
		},
	}

	SanitizeBedrockPromptCachingScope(payload)

	topLevelCache := payload["cache_control"].(map[string]interface{})
	if _, ok := topLevelCache["scope"]; ok {
		t.Fatalf("expected top-level cache_control.scope to be removed")
	}

	systemCache := payload["system"].([]interface{})[1].(map[string]interface{})["cache_control"].(map[string]interface{})
	if _, ok := systemCache["scope"]; ok {
		t.Fatalf("expected system cache_control.scope to be removed")
	}

	messageCache := payload["messages"].([]interface{})[0].(map[string]interface{})["content"].([]interface{})[0].(map[string]interface{})["cache_control"].(map[string]interface{})
	if _, ok := messageCache["scope"]; ok {
		t.Fatalf("expected message cache_control.scope to be removed")
	}

	if payload["metadata"].(map[string]interface{})["scope"] != "keep-me" {
		t.Fatalf("expected non-cache_control scope field to be preserved")
	}
}

func TestSanitizeBedrockPromptCachingBytes(t *testing.T) {
	requestBody := `{
		"model": "claude-opus-4-6",
		"system": [
			{"type": "text", "text": "base"},
			{"type": "text", "text": "cached", "cache_control": {"type": "ephemeral", "scope": "workspace"}}
		],
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "hello", "cache_control": {"type": "ephemeral", "scope": "message"}}
				]
			}
		],
		"metadata": {"scope": "keep-me"}
	}`

	sanitized, err := SanitizeBedrockPromptCachingBytes([]byte(requestBody))
	if err != nil {
		t.Fatalf("SanitizeBedrockPromptCachingBytes returned error: %v", err)
	}
	if strings.Contains(string(sanitized), `"scope":"workspace"`) || strings.Contains(string(sanitized), `"scope":"message"`) {
		t.Fatalf("expected cache_control.scope to be removed, got %s", string(sanitized))
	}
	if !strings.Contains(string(sanitized), `"metadata":{"scope":"keep-me"}`) {
		t.Fatalf("expected non-cache_control scope to be preserved, got %s", string(sanitized))
	}

	var payload map[string]any
	if err := common.Unmarshal(sanitized, &payload); err != nil {
		t.Fatalf("unmarshal sanitized body: %v", err)
	}
}

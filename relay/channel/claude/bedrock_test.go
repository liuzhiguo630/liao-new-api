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

func TestSanitizeBedrockRequestBody_ThinkingDisabled(t *testing.T) {
	body := `{
		"anthropic_version": "bedrock-2023-05-31",
		"messages": [{"role": "user", "content": "hi"}],
		"max_tokens": 1024,
		"thinking": {"type": "disabled", "budget_tokens": 0}
	}`

	sanitized, err := SanitizeBedrockRequestBody([]byte(body))
	if err != nil {
		t.Fatalf("SanitizeBedrockRequestBody error: %v", err)
	}
	if strings.Contains(string(sanitized), "budget_tokens") {
		t.Fatalf("expected budget_tokens to be removed when thinking.type=disabled, got %s", string(sanitized))
	}
	if !strings.Contains(string(sanitized), `"type":"disabled"`) {
		t.Fatalf("expected thinking.type=disabled to be preserved, got %s", string(sanitized))
	}
}

func TestSanitizeBedrockRequestBody_ThinkingEnabled(t *testing.T) {
	body := `{
		"anthropic_version": "bedrock-2023-05-31",
		"messages": [{"role": "user", "content": "hi"}],
		"max_tokens": 8192,
		"thinking": {"type": "enabled", "budget_tokens": 4096}
	}`

	sanitized, err := SanitizeBedrockRequestBody([]byte(body))
	if err != nil {
		t.Fatalf("SanitizeBedrockRequestBody error: %v", err)
	}
	if !strings.Contains(string(sanitized), "budget_tokens") {
		t.Fatalf("expected budget_tokens to be preserved when thinking.type=enabled, got %s", string(sanitized))
	}
}

func TestSanitizeBedrockRequestBody_RemovesUnsupportedFields(t *testing.T) {
	body := `{
		"anthropic_version": "bedrock-2023-05-31",
		"model": "claude-sonnet-4-20250514",
		"stream": true,
		"messages": [{"role": "user", "content": "hi"}],
		"max_tokens": 1024,
		"mcp_servers": [{"url": "http://localhost"}],
		"metadata": {"user_id": "test"},
		"service_tier": "auto",
		"inference_geo": "us",
		"context_management": {"enabled": true},
		"output_format": {"type": "json"},
		"container": {"name": "test"}
	}`

	sanitized, err := SanitizeBedrockRequestBody([]byte(body))
	if err != nil {
		t.Fatalf("SanitizeBedrockRequestBody error: %v", err)
	}
	s := string(sanitized)
	// Universal fields should be removed
	for _, field := range []string{"mcp_servers", "context_management", "output_format", "container"} {
		if strings.Contains(s, `"`+field+`"`) {
			t.Errorf("expected %s to be removed, got %s", field, s)
		}
	}
	// model, stream, metadata, service_tier, inference_geo must be PRESERVED
	// (proxied Bedrock channels need them; only AWS SDK path strips them via extra args)
	for _, field := range []string{"model", "stream", "metadata", "service_tier", "inference_geo"} {
		if !strings.Contains(s, `"`+field+`"`) {
			t.Errorf("expected %s to be preserved for proxy path, got %s", field, s)
		}
	}
}

func TestSanitizeBedrockRequestBody_ExtraFields(t *testing.T) {
	body := `{
		"model": "claude-sonnet-4-20250514",
		"stream": true,
		"messages": [{"role": "user", "content": "hi"}],
		"metadata": {"user_id": "test"},
		"service_tier": "auto",
		"inference_geo": "us"
	}`

	sanitized, err := SanitizeBedrockRequestBody([]byte(body),
		"model", "stream", "metadata", "service_tier", "inference_geo")
	if err != nil {
		t.Fatalf("SanitizeBedrockRequestBody error: %v", err)
	}
	s := string(sanitized)
	for _, field := range []string{"model", "stream", "metadata", "service_tier", "inference_geo"} {
		if strings.Contains(s, `"`+field+`"`) {
			t.Errorf("expected %s to be removed via extra args, got %s", field, s)
		}
	}
	if !strings.Contains(s, "messages") {
		t.Fatalf("expected messages to be preserved, got %s", s)
	}
}

func TestSanitizeBedrockRequestBody_NoThinking(t *testing.T) {
	body := `{
		"anthropic_version": "bedrock-2023-05-31",
		"messages": [{"role": "user", "content": "hi"}],
		"max_tokens": 1024
	}`

	sanitized, err := SanitizeBedrockRequestBody([]byte(body))
	if err != nil {
		t.Fatalf("SanitizeBedrockRequestBody error: %v", err)
	}
	if strings.Contains(string(sanitized), "thinking") {
		t.Fatalf("expected no thinking field, got %s", string(sanitized))
	}
}

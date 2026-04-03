package aws

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
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

	sanitizeBedrockPromptCachingScope(payload)

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

func TestFormatRequestRemovesUnsupportedCacheControlScopeWhenFilterEnabled(t *testing.T) {
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
		"max_tokens": 1024
	}`

	formatted, err := formatRequest(strings.NewReader(requestBody), nil, true)
	if err != nil {
		t.Fatalf("formatRequest returned error: %v", err)
	}

	encoded, err := common.Marshal(formatted)
	if err != nil {
		t.Fatalf("marshal formatted request: %v", err)
	}
	if strings.Contains(string(encoded), `"scope"`) {
		t.Fatalf("expected formatted request to strip cache_control.scope, got %s", string(encoded))
	}
}

func TestFormatRequestKeepsScopeWhenFilterDisabled(t *testing.T) {
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
		"max_tokens": 1024
	}`

	formatted, err := formatRequest(strings.NewReader(requestBody), nil, false)
	if err != nil {
		t.Fatalf("formatRequest returned error: %v", err)
	}

	encoded, err := common.Marshal(formatted)
	if err != nil {
		t.Fatalf("marshal formatted request: %v", err)
	}
	if !strings.Contains(string(encoded), `"scope"`) {
		t.Fatalf("expected formatted request to preserve cache_control.scope when filter is disabled, got %s", string(encoded))
	}
}

func TestBuildAwsRequestBodyRemovesScopeForStructuredRequestWhenFilterEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := &AwsClaudeRequest{
		System: []dto.ClaudeMediaMessage{
			{
				Type:         "text",
				Text:         common.GetPointer("base"),
				CacheControl: json.RawMessage(`{"type":"ephemeral","scope":"workspace"}`),
			},
		},
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type:         "text",
						Text:         common.GetPointer("hello"),
						CacheControl: json.RawMessage(`{"type":"ephemeral","scope":"message"}`),
					},
				},
			},
		},
	}

	body, err := buildAwsRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelOtherSettings: dto.ChannelOtherSettings{
				FilterBedrockBeta: true,
			},
		},
	}, req)
	if err != nil {
		t.Fatalf("buildAwsRequestBody returned error: %v", err)
	}
	if strings.Contains(string(body), `"scope"`) {
		t.Fatalf("expected buildAwsRequestBody to strip cache_control.scope, got %s", string(body))
	}
}

func TestBuildAwsRequestBodyKeepsScopeForStructuredRequestWhenFilterDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := &AwsClaudeRequest{
		System: []dto.ClaudeMediaMessage{
			{
				Type:         "text",
				Text:         common.GetPointer("base"),
				CacheControl: json.RawMessage(`{"type":"ephemeral","scope":"workspace"}`),
			},
		},
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type:         "text",
						Text:         common.GetPointer("hello"),
						CacheControl: json.RawMessage(`{"type":"ephemeral","scope":"message"}`),
					},
				},
			},
		},
	}

	body, err := buildAwsRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}, req)
	if err != nil {
		t.Fatalf("buildAwsRequestBody returned error: %v", err)
	}
	if !strings.Contains(string(body), `"scope"`) {
		t.Fatalf("expected buildAwsRequestBody to preserve cache_control.scope when filter is disabled, got %s", string(body))
	}
}

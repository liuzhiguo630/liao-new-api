package claude

import "github.com/QuantumNous/new-api/common"

func SanitizeBedrockPromptCachingBytes(rawBody []byte) ([]byte, error) {
	var payload any
	if err := common.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}
	SanitizeBedrockPromptCachingScope(payload)
	return common.Marshal(payload)
}

func SanitizeBedrockPromptCachingScope(payload any) {
	sanitizeBedrockPromptCachingValue(payload, false)
}

func sanitizeBedrockPromptCachingValue(value interface{}, insideCacheControl bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		if insideCacheControl {
			delete(v, "scope")
		}
		for key, child := range v {
			sanitizeBedrockPromptCachingValue(child, insideCacheControl || key == "cache_control")
		}
	case []interface{}:
		for _, child := range v {
			sanitizeBedrockPromptCachingValue(child, insideCacheControl)
		}
	default:
	}
}

// bedrockUnsupportedTopLevelFields are Claude-only fields that Bedrock never accepts.
// These are safe to strip for both direct AWS SDK calls and proxied Bedrock channels.
// NOTE: model, stream, metadata, service_tier, inference_geo are NOT here because
// proxied channels (type=14 with filter_bedrock_beta) still need them.
var bedrockUnsupportedTopLevelFields = []string{
	"prompt", "max_tokens_to_sample",
	"context_management", "output_format",
	"container", "mcp_servers",
}

// bedrockUnsupportedToolFields are per-tool definition fields that first-party
// Anthropic accepts but Bedrock rejects with "Extra inputs are not permitted"
// (loc: tools[].custom.<field>). Claude clients (e.g. Claude Code) set these on
// custom tools; we strip them from every entry of the top-level "tools" array.
//   - eager_input_streaming: opt-in for fine-grained tool streaming, Anthropic-only
var bedrockUnsupportedToolFields = []string{
	"eager_input_streaming",
}

// SanitizeBedrockRequestBody removes/fixes fields in the request body that are
// incompatible with the AWS Bedrock Claude API:
//   - Removes budget_tokens from thinking when type != "enabled" (Bedrock rejects extra fields)
//   - Removes top-level fields that Bedrock does not support
//   - Removes per-tool fields that Bedrock does not support (e.g. eager_input_streaming)
//   - extraFieldsToRemove: additional fields to strip (e.g. "model","stream" for AWS SDK path)
func SanitizeBedrockRequestBody(rawBody []byte, extraFieldsToRemove ...string) ([]byte, error) {
	var payload map[string]interface{}
	if err := common.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}
	sanitizeBedrockThinking(payload)
	sanitizeBedrockTools(payload)
	for _, field := range bedrockUnsupportedTopLevelFields {
		delete(payload, field)
	}
	for _, field := range extraFieldsToRemove {
		delete(payload, field)
	}
	return common.Marshal(payload)
}

// sanitizeBedrockThinking strips budget_tokens from thinking when type is not "enabled",
// because Bedrock's discriminated union schema rejects extra fields for other types.
func sanitizeBedrockThinking(data map[string]interface{}) {
	thinkingRaw, ok := data["thinking"]
	if !ok {
		return
	}
	thinking, ok := thinkingRaw.(map[string]interface{})
	if !ok {
		return
	}
	thinkingType, _ := thinking["type"].(string)
	if thinkingType != "enabled" {
		delete(thinking, "budget_tokens")
	}
}

// sanitizeBedrockTools removes per-tool fields that Bedrock does not accept from
// each entry of the top-level "tools" array. Bedrock validates custom tools against
// a stricter schema than first-party Anthropic and rejects unknown fields such as
// eager_input_streaming (fine-grained tool streaming) with a 400 ValidationException.
func sanitizeBedrockTools(data map[string]interface{}) {
	toolsRaw, ok := data["tools"]
	if !ok {
		return
	}
	tools, ok := toolsRaw.([]interface{})
	if !ok {
		return
	}
	for _, toolRaw := range tools {
		tool, ok := toolRaw.(map[string]interface{})
		if !ok {
			continue
		}
		for _, field := range bedrockUnsupportedToolFields {
			delete(tool, field)
		}
	}
}

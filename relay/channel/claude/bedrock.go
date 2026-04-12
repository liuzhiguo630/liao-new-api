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

// SanitizeBedrockRequestBody removes/fixes fields in the request body that are
// incompatible with the AWS Bedrock Claude API:
//   - Removes budget_tokens from thinking when type != "enabled" (Bedrock rejects extra fields)
//   - Removes top-level fields that Bedrock does not support
//   - Removes cache_control.scope (delegates to SanitizeBedrockPromptCachingScope)
//   - extraFieldsToRemove: additional fields to strip (e.g. "model","stream" for AWS SDK path)
func SanitizeBedrockRequestBody(rawBody []byte, extraFieldsToRemove ...string) ([]byte, error) {
	var payload map[string]interface{}
	if err := common.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}
	sanitizeBedrockThinking(payload)
	for _, field := range bedrockUnsupportedTopLevelFields {
		delete(payload, field)
	}
	for _, field := range extraFieldsToRemove {
		delete(payload, field)
	}
	SanitizeBedrockPromptCachingScope(payload)
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

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

// bedrockUnsupportedTopLevelFields lists Claude API fields that Bedrock does not accept.
var bedrockUnsupportedTopLevelFields = []string{
	"model", "stream", "prompt", "max_tokens_to_sample",
	"inference_geo", "context_management", "output_format",
	"container", "mcp_servers", "metadata", "service_tier",
}

// SanitizeBedrockRequestBody removes/fixes fields in the request body that are
// incompatible with the AWS Bedrock Claude API:
//   - Removes budget_tokens from thinking when type != "enabled" (Bedrock rejects extra fields)
//   - Removes top-level fields that Bedrock does not support
//   - Removes cache_control.scope (delegates to SanitizeBedrockPromptCachingScope)
func SanitizeBedrockRequestBody(rawBody []byte) ([]byte, error) {
	var payload map[string]interface{}
	if err := common.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}
	sanitizeBedrockThinking(payload)
	for _, field := range bedrockUnsupportedTopLevelFields {
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

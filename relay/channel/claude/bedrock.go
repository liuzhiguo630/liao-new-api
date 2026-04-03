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

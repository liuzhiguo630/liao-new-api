package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func TestCanReuseOriginalGeminiBody(t *testing.T) {
	originalThinkingEnabled := model_setting.GetGeminiSettings().ThinkingAdapterEnabled
	defer func() {
		model_setting.GetGeminiSettings().ThinkingAdapterEnabled = originalThinkingEnabled
	}()

	model_setting.GetGeminiSettings().ThinkingAdapterEnabled = false

	baseInfo := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiType:           constant.APITypeGemini,
			ParamOverride:     map[string]interface{}{},
			ChannelSetting:    dto.ChannelSettings{},
			UpstreamModelName: "gemini-2.5-pro",
		},
	}
	baseRequest := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello"},
				},
			},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			ThinkingConfig: &dto.GeminiThinkingConfig{
				ThinkingBudget: func() *int { v := 0; return &v }(),
			},
		},
	}

	if !canReuseOriginalGeminiBody(baseInfo, baseRequest) {
		t.Fatalf("expected original Gemini body to be reusable in the safe native path")
	}

	t.Run("reject non gemini api type", func(t *testing.T) {
		info := *baseInfo
		meta := *baseInfo.ChannelMeta
		meta.ApiType = constant.APITypeOpenAI
		info.ChannelMeta = &meta
		if canReuseOriginalGeminiBody(&info, baseRequest) {
			t.Fatalf("expected reuse to be disabled for non-gemini upstream")
		}
	})

	t.Run("reject param override", func(t *testing.T) {
		info := *baseInfo
		meta := *baseInfo.ChannelMeta
		meta.ParamOverride = map[string]interface{}{"temperature": 0.1}
		info.ChannelMeta = &meta
		if canReuseOriginalGeminiBody(&info, baseRequest) {
			t.Fatalf("expected reuse to be disabled when param override exists")
		}
	})

	t.Run("reject system prompt injection", func(t *testing.T) {
		info := *baseInfo
		meta := *baseInfo.ChannelMeta
		meta.ChannelSetting = dto.ChannelSettings{SystemPrompt: "system"}
		info.ChannelMeta = &meta
		if canReuseOriginalGeminiBody(&info, baseRequest) {
			t.Fatalf("expected reuse to be disabled when system prompt is configured")
		}
	})

	t.Run("reject native normalization cases", func(t *testing.T) {
		req := &dto.GeminiChatRequest{
			Contents: []dto.GeminiChatContent{
				{
					Parts: []dto.GeminiPart{{Text: "hello"}},
				},
			},
		}
		if canReuseOriginalGeminiBody(baseInfo, req) {
			t.Fatalf("expected reuse to be disabled when first Gemini content role is empty")
		}
	})

	t.Run("reject thinking adaptor mutation", func(t *testing.T) {
		model_setting.GetGeminiSettings().ThinkingAdapterEnabled = true
		defer func() {
			model_setting.GetGeminiSettings().ThinkingAdapterEnabled = false
		}()

		req := &dto.GeminiChatRequest{
			Contents: []dto.GeminiChatContent{
				{
					Role:  "user",
					Parts: []dto.GeminiPart{{Text: "hello"}},
				},
			},
		}
		if canReuseOriginalGeminiBody(baseInfo, req) {
			t.Fatalf("expected reuse to be disabled when thinking adaptor would mutate request")
		}
	})
}

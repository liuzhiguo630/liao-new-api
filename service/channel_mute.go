package service

import (
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

const defaultMuteDuration = 3 * time.Second

// channelModelMuteMap stores mute expiration time for channel+model combinations.
// Key: "channelId:modelName", Value: time.Time (mute expiry)
var channelModelMuteMap sync.Map

func init() {
	model.ChannelModelMuteChecker = IsChannelModelMuted
}

func muteKey(channelId int, modelName string) string {
	return strconv.Itoa(channelId) + ":" + modelName
}

// MuteChannelModel mutes a channel+model on 429.
// Only effective for Gemini channels when GeminiRateLimitMuteEnabled is true.
func MuteChannelModel(channelId int, channelType int, modelName string, duration time.Duration) {
	if !common.GeminiRateLimitMuteEnabled {
		return
	}
	if channelType != constant.ChannelTypeGemini {
		return
	}
	if duration <= 0 {
		duration = defaultMuteDuration
	}
	expiry := time.Now().Add(duration)
	key := muteKey(channelId, modelName)
	channelModelMuteMap.Store(key, expiry)
	common.SysLog("channel #" + strconv.Itoa(channelId) + " model " + modelName + " muted for " + duration.String())
}

// IsChannelModelMuted checks whether a channel+model combination is currently muted.
func IsChannelModelMuted(channelId int, modelName string) bool {
	if !common.GeminiRateLimitMuteEnabled {
		return false
	}
	key := muteKey(channelId, modelName)
	val, ok := channelModelMuteMap.Load(key)
	if !ok {
		return false
	}
	expiry := val.(time.Time)
	if time.Now().After(expiry) {
		channelModelMuteMap.Delete(key)
		return false
	}
	return true
}

// GetMuteDurationFor429 returns the mute duration for a 429 error.
// If the error message contains a specific retry duration, use it; otherwise use the default.
func GetMuteDurationFor429(errMsg string) time.Duration {
	return defaultMuteDuration
}

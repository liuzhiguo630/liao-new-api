package service

import (
	"regexp"
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

var retryInRegex = regexp.MustCompile(`(?i)retry\s+in\s+(\d+(?:\.\d+)?)\s*s`)

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

// ParseRetryDuration parses a retry duration from an error message.
// It looks for patterns like "retry in 10.459224242s" or "retry in 5s".
// Returns 0 if no retry duration is found.
func ParseRetryDuration(errMsg string) time.Duration {
	matches := retryInRegex.FindStringSubmatch(errMsg)
	if len(matches) < 2 {
		return 0
	}
	seconds, err := strconv.ParseFloat(matches[1], 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

// GetMuteDurationFor429 returns the mute duration for a 429 error.
// If the error message contains a specific retry duration, use it; otherwise use the default.
func GetMuteDurationFor429(errMsg string) time.Duration {
	d := ParseRetryDuration(errMsg)
	if d > 0 {
		return d
	}
	return defaultMuteDuration
}

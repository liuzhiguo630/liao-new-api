package vertex

import "one-api/common"

func GetModelRegion(other string, localModelName string) string {
	// if other is json string
	if common.IsJsonStr(other) {
		m := common.StrToMap(other)
		region := m[localModelName]
		if region == nil {
			region = m["default"]
		}
		// 如果是 string 则直接返回，否则可能是 array 随机挑一个返回
		if s, ok := region.(string); ok {
			return s
		}
		if arr, ok := region.([]interface{}); ok && len(arr) > 0 {
			// Assuming common package provides RandSelectString for random selection and conversion
			return arr[common.GetRandomInt(len(arr))].(string)
		}
	}
	return other
}

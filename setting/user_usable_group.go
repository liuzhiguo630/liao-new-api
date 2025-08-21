package setting

import (
	"encoding/json"
	"one-api/common"
	"sync"
)

var (
	userUsableGroups = map[string]string{
		"default": "默认分组",
		"vip":     "vip分组",
	}
	userUsableGroupsMutex sync.RWMutex
)

func GetUserUsableGroupsCopy() map[string]string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	copyUserUsableGroups := make(map[string]string)
	for k, v := range userUsableGroups {
		copyUserUsableGroups[k] = v
	}
	return copyUserUsableGroups
}

func UserUsableGroups2JSONString() string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	jsonBytes, err := json.Marshal(userUsableGroups)
	if err != nil {
		common.SysError("error marshalling user groups: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateUserUsableGroupsByJSONString(jsonStr string) error {
	newGroups := make(map[string]string)
	err := json.Unmarshal([]byte(jsonStr), &newGroups)
	if err != nil {
		return err
	}

	userUsableGroupsMutex.Lock()
	userUsableGroups = newGroups
	userUsableGroupsMutex.Unlock()

	return nil
}

func GetUserUsableGroups(userGroup string) map[string]string {
	groupsCopy := GetUserUsableGroupsCopy()
	if userGroup == "" {
		if _, ok := groupsCopy["default"]; !ok {
			groupsCopy["default"] = "default"
		}
	}
	// 如果userGroup不在UserUsableGroups中，返回UserUsableGroups + userGroup
	if _, ok := groupsCopy[userGroup]; !ok {
		groupsCopy[userGroup] = "用户分组"
	}
	// 如果userGroup在UserUsableGroups中，返回UserUsableGroups
	return groupsCopy
}

func GroupInUserUsableGroups(groupName string) bool {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	_, ok := userUsableGroups[groupName]
	return ok
}

func GetUsableGroupDescription(groupName string) string {
	userUsableGroupsMutex.RLock()
	defer userUsableGroupsMutex.RUnlock()

	if desc, ok := userUsableGroups[groupName]; ok {
		return desc
	}
	return groupName
}

package util

import "encoding/json"

func GetMapValueToInt(data any, key string) int {
	pMap, pBool := data.(map[string]any)
	if pBool {
		v, b := pMap[key]
		if b {
			v2, b2 := v.(float64)
			if b2 {
				return int(v2)
			}
			return 0
		}
		return 0
	} else {
		return 0
	}
}

func GetMapValueToString(data any, key string) string {
	pMap, pBool := data.(map[string]any)
	if pBool {
		v, b := pMap[key]
		if b {
			v2, b2 := v.(string)
			if b2 {
				return v2
			}
			return ""
		}
		return ""
	} else {
		return ""
	}
}

func SetJsonTo(data any, to any) bool {
	j, b := json.Marshal(data)
	if b == nil {
		e := json.Unmarshal(j, &to)
		if e == nil {
			return true
		} else {
			return false
		}
	}
	return false
}

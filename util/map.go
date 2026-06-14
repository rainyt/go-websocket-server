package util

import (
	"encoding/json"
	"sync"
)

type Map struct {
	lock sync.Mutex
	data map[string]any
}

func (m *Map) Copy() map[string]any {
	m.lock.Lock()
	defer m.lock.Unlock()
	v, err := json.Marshal(m.data)
	if err != nil {
		return nil
	}
	map2 := &map[string]any{}
	err2 := json.Unmarshal(v, map2)
	if err2 == nil {
		return *map2
	} else {
		return nil
	}
}

func CreateMap() *Map {
	return &Map{
		data: map[string]any{},
	}
}

func (m *Map) Store(key string, data any) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.data[key] = data
}

func (m *Map) GetData(key string, data any) any {
	m.lock.Lock()
	defer m.lock.Unlock()
	v, b := m.data[key]
	if b {
		return v
	}
	return data
}

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

func GetMapValueToAny(data any, key string) any {
	pMap, pBool := data.(map[string]any)
	if pBool {
		v, b := pMap[key]
		if b {
			return v
		}
		return nil
	} else {
		return nil
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

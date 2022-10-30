package util

import "sync"

type Array struct {
	lock sync.Mutex
	List []any `json:"list"`
}

func CreateArray() *Array {
	return &Array{
		List: []any{},
	}
}

// 添加数组
func (a *Array) Push(o any) {
	a.lock.Lock()
	a.List = append(a.List, o)
	a.lock.Unlock()
}

// 从数组中删除
func (a *Array) Remove(o any) bool {
	a.lock.Lock()
	for i := 0; i < len(a.List); i++ {
		if a.List[i] == o {
			a.List = append(a.List[0:i], a.List[i+1:]...)
			a.lock.Unlock()
			return true
		}
	}
	a.lock.Unlock()
	return false
}

// 获取数组长度
func (a *Array) Length() int {
	return len(a.List)
}

// 获取位置，如果不存在则会返回-1
func (a *Array) IndexOf(o any) int {
	for i := 0; i < len(a.List); i++ {
		if a.List[i] == o {
			return i
		}
	}
	return -1
}

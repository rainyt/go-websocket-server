package util

type Array struct {
	List []any
}

// 添加数组
func (a *Array) Push(o any) {
	a.List = append(a.List, o)
}

// 从数组中删除
func (a *Array) Remove(o any) {
	for i := 0; i < len(a.List); i++ {
		if a.List[i] == o {
			a.List = append(a.List[0:i], a.List[i+1:]...)
			break
		}
	}
}

// 获取数组长度
func (a *Array) Length() int {
	return len(a.List)
}

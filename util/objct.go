package util

type Object struct {
	Data any
}

func CreateObject(data any) *Object {
	return &Object{
		Data: data,
	}
}

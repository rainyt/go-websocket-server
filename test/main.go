package main

import (
	"fmt"
	"websocket_server/util"
)

func main() {
	array := util.Array{
		List: []any{},
	}
	array.Push(1)
	array.Push(2)
	array.Push(3)
	array.Push(4)
	array.Push(5)
	array.Remove(3)
	fmt.Print(array)
}

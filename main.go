package main

import "github.com/Shopify/go-lua"

func main() {
	Init()
	l := lua.NewState()
	lua.OpenLibraries(l)
	s:=`return {1,2,3}`
	lua.LoadString(l, s)
	l.Call(0, 1)
	l.SetGlobal("tt")
	if err := lua.DoFile(l, "logic/lua/init.lua"); err != nil {
		panic(err)
	}

}

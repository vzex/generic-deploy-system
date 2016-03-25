package common

import "github.com/yuin/gopher-lua"
import "os/exec"
import "strings"

func RegLuaFunc(l *lua.LState, name string, f func(l *lua.LState) int) {
        l.SetGlobal(name, l.NewFunction(f))
}

func RegLuaFuncWithCancel(l *lua.LState, name string, f func(l *lua.LState, sessionQuitC chan bool) int, sessionQuitC chan bool) {
        l.SetGlobal(name, l.NewFunction(func(l *lua.LState) int {
                return f(l, sessionQuitC)
        }))
}
func InitCommon(l *lua.LState, sessionQuitC chan bool) {
        RegLuaFuncWithCancel(l, "cmd", cmd, sessionQuitC)
        RegLuaFuncWithCancel(l, "bash", bash, sessionQuitC)
}

func cmd(l *lua.LState, sessionQuitC chan bool) int {
        c := l.CheckString(1)
        arr:=strings.Split(c, " ")
        if len(arr) > 0 {
		out := ""
		ch:=make(chan string)
		bok := true
		cmd := exec.Command(arr[0], arr[1:]...)
		go func() {
			msg, err := cmd.CombinedOutput()
			if err != nil {
				bok = false
			}
			select {
			case ch<-string(msg):
			case <-sessionQuitC:
			}
		}()
		select {
		case out = <-ch:
		case <-sessionQuitC:
			if cmd.Process!=nil {
				cmd.Process.Kill()
			}
		}
                l.Push(lua.LString(out))
                l.Push(lua.LBool(bok))
        } else {
                l.Push(lua.LString(""))
                l.Push(lua.LBool(false))
        }
        return 2
}

func bash(l *lua.LState, sessionQuitC chan bool) int {
        c := l.CheckString(1)
	out := ""
	ch:=make(chan string)
	bok := true
        cmd := exec.Command("/bin/bash", "-c", c)
	go func() {
		msg, err := cmd.CombinedOutput()
		if err != nil {
			bok = false
		}
		select {
		case ch<-string(msg):
		case <-sessionQuitC:
		}
	}()
	select {
	case out = <-ch:
	case <-sessionQuitC:
		if cmd.Process!=nil {
			cmd.Process.Kill()
		}
	}
        l.Push(lua.LString(out))
        l.Push(lua.LBool(bok))
        return 2
}

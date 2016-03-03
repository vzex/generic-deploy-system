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
}

func cmd(l *lua.LState, sessionQuitC chan bool) int {
        c := l.CheckString(1)
        arr:=strings.Split(c, " ")
        if len(arr) > 0 {
                out, err := exec.Command(arr[0], arr[1:]...).CombinedOutput()
                bok := true
                if err != nil {
                        bok = false
                }
                l.Push(lua.LString(out))
                l.Push(lua.LBool(bok))
        } else {
                l.Push(lua.LString(""))
                l.Push(lua.LBool(false))
        }
        return 2
}

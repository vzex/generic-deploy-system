package common

import "github.com/yuin/gopher-lua"
import "os/exec"
import "strings"

func RegLuaFunc(l *lua.LState, name string, f func(l *lua.LState) int) {
        l.SetGlobal(name, l.NewFunction(f))
}

func InitCommon(l *lua.LState) {
        RegLuaFunc(l, "cmd", cmd)
}

func cmd(l *lua.LState) int {
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

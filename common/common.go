package common

import "github.com/yuin/gopher-lua"
import "os/exec"
import "net"
import "encoding/base64"
import "bufio"
import "time"
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
        RegLuaFuncWithCancel(l, "connect", connect, sessionQuitC)
        RegLuaFuncWithCancel(l, "base64", func(l *lua.LState, quit chan bool) int {
                l.Push(lua.LString(base64.StdEncoding.EncodeToString([]byte(l.CheckString(1)))))
                return 1
        }, sessionQuitC)
        RegLuaFuncWithCancel(l, "from64", func(l *lua.LState, quit chan bool) int {
                s, _ := base64.StdEncoding.DecodeString(l.CheckString(1))
                l.Push(lua.LString(s))
                return 1
        }, sessionQuitC)
}

func connect(l *lua.LState, sessionQuitC chan bool) int {
        addr := l.CheckString(1)
        timeout := l.CheckInt(2)
        conn, er := net.DialTimeout("tcp", addr, time.Duration(timeout)*time.Second)
        if er != nil {
                l.Push(lua.LString(er.Error()))
                return 1
        }
        call := l.Get(3)
        if call.Type() != lua.LTFunction {
                return 0
        }
        f:=func(l *lua.LState) int {
                s := l.CheckString(1)
                switch s {
                case "close":
                        conn.Close()
                        return 0
                case "send":
                        conn.Write([]byte(l.CheckString(2)))
                }
                return 0
        }
        go func() {
                for {
                        select {
                        case <- sessionQuitC:
                                conn.Close()
                                return
                        }
                }
        }()
        newf:=l.NewFunction(f)
        l.Push(newf)
        l.Push(lua.LString("connected"))
        if e := l.PCall(2, 0, nil); e != nil {
                l.Push(lua.LString(e.Error()))
                return 1
        }
        scanner := bufio.NewScanner(conn)
        scanner.Split(func(data []byte, atEOF bool) (adv int, token []byte, err error) {
                if len(data) <= 0 {
                        return 0, nil, nil
                }
                l.Push(call)
                l.Push(newf)
                l.Push(lua.LString("recv"))
                l.Push(lua.LString(data))
                if e := l.PCall(3, 0, nil); e != nil {
                        conn.Close()
                        return 0, nil, e
                }
                return len(data), []byte{}, nil
        })
        for scanner.Scan() {
        }
        if scanner.Err() != nil {
                l.Push(lua.LString(scanner.Err().Error()))
                return 1
        }
        return 0
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

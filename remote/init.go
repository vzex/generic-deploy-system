package remote
import "flag"
import "log"
import "net"
import "../pipe"
import "github.com/yuin/gopher-lua"

var service = flag.String("service", "127.0.0.1:8888", "")
var groupName = flag.String("group", "", "")
var nickName = flag.String("nick", "", "")
func Init() {
	flag.Parse()
	println("remote init")
        c:=make(chan *pipe.HelperInfo)
	d:=make(chan bool)
        go func() {
                for {
                        select {
                        case info:=<-c:
                                switch info.Cmd {
				case pipe.Shutdown:
					d<-true
                                case pipe.Request:
					var s pipe.RequestCmd
					pipe.DecodeBytes(info.Bytes, &s)
					handleRequest(s, info.Conn)
                                }
                        }
                }
        }()
        client := pipe.NewInnerClient(*service, c)
        if client!=nil {
		info := &pipe.RemoteInfo{}
		info.Group = *groupName
		info.Nick = *nickName
               client.Send(pipe.RegRemote, info)
	       <-d
        } else {
		log.Println("dial fail")
	}
}

func handleRequest(s pipe.RequestCmd, conn net.Conn) {
	l := lua.NewState()
	l.OpenLibs()
	if err := l.DoFile("logic/internal/init.lua"); err != nil {
		log.Println(err.Error())
		return //todo
	}
	l.SetGlobal("MsgPack", l.Get(-1))
	l.Pop(1)
	id := s.Id
	RegLuaFunc(l, "SendToRemote", func(l *lua.LState) int {
		return SendToRemote(int(id), l, conn, "recv")
	})
	RegLuaFunc(l, "SendToRemoteEnd", func(l *lua.LState) int {
		return SendToRemote(int(id), l, conn, "end")
	})
	str:=s.Cmd
	if _err := l.DoFile("logic_remote/handle.lua"); _err != nil { 
		log.Println(_err.Error())
		return //todo
	}
	l.Push(l.GetGlobal("_handle"))
	l.Push(lua.LString(str))
	l.Call(1, 0)
	l.Push(lua.LString(""))
	SendToRemote(int(id), l, conn, "end")
}

func SendToRemote(requestid int, l *lua.LState, conn net.Conn, t string) int {
	s := l.CheckString(1)
	k := &pipe.ResponseCmd{uint(requestid), s, t}
	pipe.Send(conn, pipe.Response, k)
	return 0
}
func RegLuaFunc(l *lua.LState, name string, f func(l *lua.LState) int) {
	l.SetGlobal(name, l.NewFunction(f))
}

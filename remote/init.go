package remote
import "flag"
import "log"
import "net"
import "../pipe"
import "github.com/yuin/gopher-lua"
import "../common"

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
					go handleRequest(s, info.Conn)
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
        common.InitCommon(l)
	if err := l.DoFile("logic_remote/internal/init.lua"); err != nil {
		log.Println(err.Error())
		return //todo
	}
	l.SetGlobal("MsgPack", l.Get(-1))
	l.Pop(1)
	id := s.Id
	common.RegLuaFunc(l, "SendBack", func(l *lua.LState) int {
		return SendBack(int(id), l, conn, "recv")
	})
	common.RegLuaFunc(l, "SendBackEnd", func(l *lua.LState) int {
		return SendBack(int(id), l, conn, "end")
	})
	str:=s.Cmd
	if _err := l.DoFile("logic_remote/handle.lua"); _err != nil { 
		log.Println(_err.Error())
		return //todo
	}
        mp:=l.GetGlobal("MsgPack")
        l.Push(mp.(*lua.LTable).RawGetString("unpack"))
	l.Push(lua.LString(str))

	if e := l.PCall(1, 1, nil); e != nil {
		log.Println(e.Error())
	}
        t:=l.Get(-1)
        action := "handle_"  + string(t.(*lua.LTable).RawGetString("Action").(lua.LString))
        l.Push(l.GetGlobal(action))
	l.Push(t)
	if e := l.PCall(1, 0, nil); e != nil {
		log.Println("call error", e.Error())
	}
        l.SetTop(0)
        l.Push(lua.LString(""))
	SendBack(int(id), l, conn, "end")
}

func SendBack(requestid int, l *lua.LState, conn net.Conn, t string) int {
	s := l.CheckString(1)
	k := &pipe.ResponseCmd{uint(requestid), s, t}
	pipe.Send(conn, pipe.Response, k)
	return 0
}

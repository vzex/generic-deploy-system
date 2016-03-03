package remote
import "flag"
import "log"
import "net"
import "sync"
import "../pipe"
import "github.com/yuin/gopher-lua"
import "../common"

var service = flag.String("service", "127.0.0.1:8888", "")
var groupName = flag.String("group", "", "")
var nickName = flag.String("nick", "", "")

type sessionInfo struct {
        quitC chan bool
}

type SessionMgr struct {
        tbl map[int]*sessionInfo
        sync.RWMutex
}

func (s *SessionMgr) AddSession(id int) *sessionInfo {
        s.Lock()
        session := &sessionInfo{make(chan bool)}
        s.tbl[id] = session
        s.Unlock()
        return session
}
func (s *SessionMgr) GetSession(id int) *sessionInfo {
        s.RLock()
        session, _ := s.tbl[id]
        s.RUnlock()
        return session
}
func (s *SessionMgr) DelSession(id int) {
        session := s.GetSession(id)
        if session != nil {
                close(session.quitC)
        }
        s.Lock()
        delete(s.tbl, id)
        s.Unlock()
}
var g_SessionTbl *SessionMgr

func Init() {
        g_SessionTbl = &SessionMgr{tbl:make(map[int]*sessionInfo)}
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

//todo , save to table, for cancel
func handleRequest(s pipe.RequestCmd, conn net.Conn) {
        sessionid := s.SessionId
        session := g_SessionTbl.AddSession(sessionid)

	l := lua.NewState()
	l.OpenLibs()
        common.InitCommon(l, session.quitC)
	if err := l.DoFile("logic_remote/internal/init.lua"); err != nil {
		log.Println(err.Error())
		return //todo
	}
	l.SetGlobal("MsgPack", l.Get(-1))
	l.Pop(1)
	id := int(s.Id)
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
        g_SessionTbl.DelSession(sessionid)
}

func SendBack(requestid int, l *lua.LState, conn net.Conn, t string) int {
	s := l.CheckString(1)
	k := &pipe.ResponseCmd{uint(requestid), s, t}
	pipe.Send(conn, pipe.Response, k)
	return 0
}

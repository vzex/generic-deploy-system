package remote
import "flag"
import "log"
import "net"
import "io/ioutil"
import "sync"
import "../pipe"
import "github.com/yuin/gopher-lua"
import "../common"

var service = flag.String("service", "127.0.0.1:8888", "")
var groupName = flag.String("group", "", "")
var nickName = flag.String("nick", "", "")

type sessionInfo struct {
        sessionId int
        quitC chan bool
}

type SessionMgr struct {
        tbl map[int]*sessionInfo
        sessiontbl map[int](map[int]bool)
        sync.RWMutex
}

func (s *SessionMgr) AddSession(id, sessionId int) *sessionInfo {
        s.Lock()
        session := &sessionInfo{sessionId, make(chan bool)}
        s.tbl[id] = session
        ss, b := s.sessiontbl[sessionId]
        if !b {
                ss = make(map[int]bool)
                s.sessiontbl[sessionId] = ss
        }
        ss[id] = true
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
        s.Lock()
        if session != nil {
                sid := session.sessionId
                ss, b := s.sessiontbl[sid]
                if b {
                        delete(ss, id)
                        if len(ss) == 0 {
                                delete(s.sessiontbl, sid)
                        }
                }
                close(session.quitC)
        }
        delete(s.tbl, id)
        s.Unlock()
}

func (s *SessionMgr) CancelSession(sessionId int) {
        s.Lock()
        ss, b := s.sessiontbl[sessionId]
        if b {
                for id, _ := range ss {
                        session, _b := s.tbl[id]
                        if _b {
                                delete(s.tbl, id)
                                close(session.quitC)
                                log.Println("cancel request", id)
                        }
                }
                delete(s.sessiontbl, sessionId)
        }
        s.Unlock()
}

var g_SessionTbl *SessionMgr

func Init() {
        g_SessionTbl = &SessionMgr{tbl:make(map[int]*sessionInfo), sessiontbl:make(map[int](map[int]bool))}
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
                                case pipe.CancelRequest:
					var s pipe.RequestCmd
					pipe.DecodeBytes(info.Bytes, &s)
                                        log.Println("CancelSession", s.SessionId)
					go g_SessionTbl.CancelSession(s.SessionId)
                                case pipe.UploadFile:
                                        var s pipe.FileCmd
					pipe.DecodeBytes(info.Bytes, &s)
                                        go writeFile(s, info.Conn)
                                case pipe.DownloadFile:
                                        var s pipe.FileCmd
					pipe.DecodeBytes(info.Bytes, &s)
                                        go readFile(s, info.Conn)
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

func readFile(s pipe.FileCmd, conn net.Conn) {
        b, er := ioutil.ReadFile(s.Name)
        var k *pipe.ResponseCmd
        if er == nil {
                k = &pipe.ResponseCmd{uint(s.Id), string(b), ""}
        } else {
                k = &pipe.ResponseCmd{uint(s.Id), "", er.Error()}
        }

        pipe.Send(conn, pipe.Response, k)
}
func writeFile(s pipe.FileCmd, conn net.Conn) {
        er:=ioutil.WriteFile(s.Name, s.Data, 0777)
        var k *pipe.ResponseCmd
        if er == nil {
                k = &pipe.ResponseCmd{uint(s.Id), "", ""}
        } else {
                k = &pipe.ResponseCmd{uint(s.Id), er.Error(), ""}
        }
        pipe.Send(conn, pipe.Response, k)
}

func handleRequest(s pipe.RequestCmd, conn net.Conn) {
        id := int(s.Id)
        session := g_SessionTbl.AddSession(id, s.SessionId)

        l := lua.NewState()
        l.OpenLibs()
        common.InitCommon(l, session.quitC)
        if err := l.DoFile("logic_remote/internal/init.lua"); err != nil {
                log.Println(err.Error())
                return //todo
        }
        l.SetGlobal("MsgPack", l.Get(-1))
        l.Pop(1)
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
        g_SessionTbl.DelSession(id)
}

func SendBack(requestid int, l *lua.LState, conn net.Conn, t string) int {
        s := l.CheckString(1)
        k := &pipe.ResponseCmd{uint(requestid), s, t}
        pipe.Send(conn, pipe.Response, k)
        return 0
}

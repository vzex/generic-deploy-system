package server
import "net/http"
import "html/template"
import "log"
import "time"
import "../pipe"
import "bufio"
import "io/ioutil"
import "sync"
import "encoding/binary"
import "errors"
import "strconv"
import "strings"
import "golang.org/x/net/websocket"
import "encoding/json"
import "github.com/yuin/gopher-lua"
import "../common"


var currId int = 0
var genIdLock sync.RWMutex
func genId() int {
	genIdLock.Lock()
	defer genIdLock.Unlock()
	currId++
	if currId > 1073741824 {
		currId = 0
	}
	return currId
}

var requestMgr *requestMgrT
type responseT struct {
        head string
        msg string
}

func InitAdminPort(addr string) error {
        http.Handle("/css/", http.FileServer(http.Dir("website")))
        http.Handle("/fonts/", http.FileServer(http.Dir("website")))
        http.Handle("/js/", http.FileServer(http.Dir("website")))
        http.HandleFunc("/down", func(w http.ResponseWriter, r *http.Request) {
                id, _ := strconv.Atoi(r.FormValue("id"))
                defer func() {
                        q:=requestMgr.GetRequest(int(id))
                        if q == nil {
                                return
                        }
                        select {
                        case <-q.closeC:
                        case q.waitC <- responseT{}:
                        }
                }()
                q:=requestMgr.GetRequest(int(id))
                if q == nil {
                        return
                }
                w.Header().Set("Content-Disposition", "attachment; filename=\""+string(q.arg2)+"\"")
                w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
                w.Write(q.arg)
        })
        http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
                file, _, err := r.FormFile("filepath")
                if err == nil {
                        b, _ := ioutil.ReadAll(file)
                        id, _ := strconv.Atoi(r.FormValue("rid"))
                        OnRecvMsg(pipe.ResponseCmd{Action:string(b), Id:uint(id)})
                } else {
                        log.Println(err.Error())
                }

        })
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                t, er := template.ParseFiles("website/index.html")
                if er == nil {
                        t.Execute(w, addr)
                } else {
                        log.Println("err:", er.Error())
                }
        })
	http.Handle("/ws", websocket.Handler(ProcessClient))
        log.Println("begin serve http:", addr)
        go http.ListenAndServe(addr, nil)
	requestMgr = &requestMgrT{}
	requestMgr.Init()
        return nil
}

type requestT struct {
	m *Machine
	waitC chan responseT
	closeC chan bool
	id int
        arg []byte
        arg2 []byte
}
type requestMgrT struct {
	tbl map[int]*requestT
	sync.RWMutex
}
func (r *requestMgrT) Init() {
	r.tbl = make(map[int]*requestT)
}

func (r *requestMgrT) GetRequest(id int) *requestT {
	r.RLock()
	_q, _ := r.tbl[id]
	r.RUnlock()
	return _q
}

func (r *requestMgrT) AddRequest(q *requestT) {
	r.Lock()
	id := q.id
	_q, h := r.tbl[id]
	if h {
		delete(r.tbl, id)
		close(_q.closeC)
	}
	r.tbl[id] = q
	//log.Println("add req", id)
	r.Unlock()
}
func (r *requestMgrT) RemoveRequest(id int) {
	r.Lock()
	_q, h := r.tbl[id]
	if h {
		//log.Println("remove req", id)
		delete(r.tbl, id)
		close(_q.closeC)
	}
	r.Unlock()
}

func ProcessClient(ws *websocket.Conn) {
	log.Println("begin echo")
	ClientTbl.Add(ws)
        scanner := bufio.NewScanner(ws)
        scanner.Split(func(data []byte, atEOF bool) (adv int, token []byte, err error) {
                return split(data, atEOF, ws, ClientReadCallBack)
        })
        for scanner.Scan() {
        }

	log.Println("end echo")
	ClientTbl.Del(ws)
}

func WSWrite(conn *websocket.Conn, head, b []byte) {
        c := make([]byte, len(head)+len(b)+2+4)
        binary.LittleEndian.PutUint16(c, uint16(len(head)))
        binary.LittleEndian.PutUint32(c[2:], uint32(len(b)))
        copy(c[6:6+len(head)], head)
        copy(c[6+len(head):], b)
        websocket.Message.Send(conn, c)
    //    a:=int(c[0])+int(c[1]>>8)
      //  g:=int(c[2])+int(c[3]>>8)+int(c[4]>>16)+int(c[5]>>24)


        //log.Println("write", len(c), a, g, len(head), len(b))
}

type ReadCallBack func(conn *websocket.Conn, head string, arg []byte)
func split(data []byte, atEOF bool, conn *websocket.Conn, callback ReadCallBack) (adv int, token []byte, err error) {
        l := len(data)
        if l < 6 {
                return 0, nil, nil
        }
        if l > 100000 {
                conn.Close()
                log.Println("invalid query!")
                return 0, nil, errors.New("to large data!")
        }
        var len1, len2 int
        len1 = int(binary.LittleEndian.Uint16(data[:2]))
        len2 = int(binary.LittleEndian.Uint32(data[2:6]))
        offset := 0
        if len1 + len2 + 6 > l {
                conn.Close()
                log.Println("invalid data", len1, len2, l)
                return 0, nil, errors.New("invalid data")
        }
        offset += len1+len2+6
        head := string(data[6:6+len1])
        tail := data[6+len1:6+len1+len2]
        callback(conn, head, tail)
        return offset, []byte{}, nil
}

func ClickAction(file string, conn *websocket.Conn, arg string, argid int) {
        client := ClientTbl.Get(conn)
        if client == nil {return}
	ar := strings.Split(file, ":")
	if len(ar) != 2 {return}
	f, m := ar[0], ar[1]
	_ar := strings.Split(f, "/")
	if len(_ar) != 2 {return}
	group := _ar[0]
	ms := RemoteTbl.GetMachines(group)
	//log.Println("test", f, m, group)
	if ms ==nil {
		return
	}
        totalN := 0
        waitC := make(chan bool)
        action := group+":"+m+":"+f
	ca:= func(ma *Machine) {
                defer func() {
                        waitC<-true
                } ()
                session := client.AddSession(ma.conn)
		single := (ClientTbl.HasActionSession(action)<=0)
                defer client.DelSession(session.id)
		ClientTbl.AddAction(action, session)
		defer ClientTbl.RemoveAction(action, session)
		l := lua.NewState()
		l.OpenLibs()
		l.SetGlobal("MachineGroup", lua.LString(group))
		l.SetGlobal("MachineName", lua.LString(ma.Nick))
		l.SetGlobal("MachineAddr", lua.LString(ma.conn.RemoteAddr().String()))
		l.SetGlobal("ExtraArg", lua.LString(arg))
                common.InitCommon(l, session.quitC)
		common.RegLuaFunc(l, "ScanButtons", func(l *lua.LState) int {
			ScanButtons()
                        return 0
		})
		common.RegLuaFunc(l, "GetNickList", func(l *lua.LState) int {
                        t:=l.NewTable()
                        for _, m := range ms.GetAll() {
                                t.Append(lua.LString(m.Nick))
                        }
                        l.Push(t)
                        return 1
		})
		common.RegLuaFunc(l, "SendToNick", func(l *lua.LState) int {
                        c:=make(chan responseT)
			id := genId()
                        go ClickAction(_ar[0] + "/" + l.CheckString(2) + ":" + l.CheckString(1), conn, l.CheckString(3), id)
                        requestMgr.AddRequest(&requestT{id:id, m:ma,waitC:c, closeC:make(chan bool)})
                        defer requestMgr.RemoveRequest(id)
                        for {
                                select {
                                case info:= <- c:
                                        l.Push(lua.LString(info.msg))
                                        return 1
                                case <-session.quitC:
                                        return 0
                                }
                        }
                        return 0
		})
		common.RegLuaFunc(l, "SendToLocal", func(l *lua.LState) int {
                        s := l.CheckString(1)
			WSWrite(conn, []byte("output"), []byte(s))
                        return 0
		})
		common.RegLuaFunc(l, "SendToRemote", func(l *lua.LState) int {
			id := genId()
			return SendToRemote(id, session.id, session.quitC, ma, l)
		})
		common.RegLuaFunc(l, "Single", func(l *lua.LState) int {
			l.Push(lua.LBool(single))
			return 1
		})
		common.RegLuaFunc(l, "ServerUploadToRemote", func(l *lua.LState) int {
			id := genId()
			return ServerUploadToRemote(id, session.id, session.quitC, ma, l)
		})
		common.RegLuaFunc(l, "ServerDownFromRemote", func(l *lua.LState) int {
			id := genId()
			return ServerDownFromRemote(id, session.id, session.quitC, ma, l)
		})
		common.RegLuaFunc(l, "LocalUploadToServer", func(l *lua.LState) int {
			id := genId()
			return LocalUploadToServer(id, session.id, session.quitC, ma, l, conn)
		})
		common.RegLuaFunc(l, "LocalGetInput", func(l *lua.LState) int {
			id := genId()
			return LocalGetInput(id, session.id, session.quitC, ma, l, conn)
		})
		common.RegLuaFunc(l, "LocalDownFromServer", func(l *lua.LState) int {
			id := genId()
			return LocalDownFromServer(id, session.id, session.quitC, ma, l, conn)
                })
		if err := l.DoFile("logic/internal/init.lua"); err != nil {
                        log.Println("call init file fail:", err.Error())
			WSWrite(conn, []byte("error"), []byte(err.Error()))
                        return
		}
                l.SetGlobal("MsgPack", l.Get(-1))
                l.Pop(1)
		if _err := l.DoFile("logic/"+ f + ".lua"); _err != nil {
                        log.Println("call logic file fail:", _err.Error())
			WSWrite(conn, []byte("error"), []byte(_err.Error()))
		}
                if argid > 0 {
                        backarg := l.Get(-1)
                        if backarg.Type() == lua.LTString {
                                go func() {
                                        q:=requestMgr.GetRequest(int(argid))
                                        if q == nil {
                                                return
                                        }
                                        select {
                                        case <-q.closeC:
                                        case q.waitC <- responseT{"", backarg.String()}:
                                        }
                                }()
                        }
                }
	}
	if m == "all" {
		t := ms.GetAll()
		for _, machine := range t {
                        totalN += 1
			go ca(machine)
		}
	} else {
                totalN = 1
		machine := ms.Get(m)
		if machine == nil {
			return
		}
		go ca(machine)
	}
        if totalN > 0 {
                name := _ar[1]
                b, _ := json.Marshal([]string{name, action})
                WSWrite(conn, []byte("lock"), []byte(b))
                defer WSWrite(conn, []byte("unlock"), []byte(name))
                for {
                        select {
                        case <- waitC:
                                totalN -= 1
                                if totalN <= 0 {
                                        return
                                }
                        }
                }
        }
}

//-------------------------
func LocalGetInput(requestid, sessionid int, sessionQuit chan bool, ma *Machine, l *lua.LState, conn *websocket.Conn) int {
	c:=make(chan responseT)
        requestMgr.AddRequest(&requestT{id:requestid, m:ma,waitC:c, closeC:make(chan bool)})
	defer requestMgr.RemoveRequest(requestid)
        WSWrite(conn, []byte("input"), []byte(strconv.Itoa(requestid)))
	for {
		select {
                case info := <- c:
                        l.Push(lua.LString(info.head))
                        return 1
                case <-sessionQuit:
                        return 0
		}
	}
}
func LocalDownFromServer(requestid, sessionid int, sessionQuit chan bool, ma *Machine, l *lua.LState, conn *websocket.Conn) int {
        from := l.CheckString(1)
	c:=make(chan responseT)
        b, er := ioutil.ReadFile(from)
        if er != nil {
                l.Push(lua.LString(er.Error())) 
                return 1
        }
        requestMgr.AddRequest(&requestT{id:requestid, m:ma,waitC:c, closeC:make(chan bool), arg : b, arg2:[]byte(from)})
	defer requestMgr.RemoveRequest(requestid)
        WSWrite(conn, []byte("downfile"), []byte(strconv.Itoa(requestid)))
	for {
		select {
		case <- c:
                        return 0
                case <-sessionQuit:
                        l.Push(lua.LString("quit"))
                        return 1
		}
	}
}

func LocalUploadToServer(requestid, sessionid int, sessionQuit chan bool, ma *Machine, l *lua.LState, conn *websocket.Conn) int {
        to := l.CheckString(1)
	c:=make(chan responseT)
	requestMgr.AddRequest(&requestT{id:requestid, m:ma,waitC:c, closeC:make(chan bool)})
	defer requestMgr.RemoveRequest(requestid)
        WSWrite(conn, []byte("uploadfile"), []byte(strconv.Itoa(requestid)))
	for {
		select {
		case info:= <- c:
                        b:=[]byte(info.head)
                        if er := ioutil.WriteFile(to, b, 0777); er == nil {
                                WSWrite(conn, []byte("uploadfileres"), []byte("1"))
                                return 0
                        } else {
                                WSWrite(conn, []byte("uploadfileres"), []byte("0"))
                                log.Println(er.Error())
                                l.Push(lua.LString(er.Error())) 
                                return 1
                        }
                case <-sessionQuit:
                        l.Push(lua.LString("quit"))
                        WSWrite(conn, []byte("uploadfileres"), []byte("0"))
                        return 1
		}
	}
}

func ServerDownFromRemote(requestid, sessionid int, sessionQuit chan bool, ma *Machine, l *lua.LState) int {
        from := l.CheckString(1)
        to := l.CheckString(2)
        t := &pipe.FileCmd{requestid, from, []byte{}}
	c:=make(chan responseT)
	requestMgr.AddRequest(&requestT{id:requestid, m:ma,waitC:c, closeC:make(chan bool)})
	defer requestMgr.RemoveRequest(requestid)
	pipe.Send(ma.conn, pipe.DownloadFile, t)
	for {
		select {
		case info:= <- c:
                        if l.Get(3).Type() == lua.LTFunction {
                                l.Push(l.Get(3))
                                if info.head == "" {
                                        er:=ioutil.WriteFile(to, []byte(info.msg), 0777)
                                        log.Println("writefile", len(info.msg), to)
                                        if er != nil {
                                                l.Push(lua.LString(er.Error()))
                                        } else {
                                                l.Push(lua.LString(""))
                                        }
                                } else {
                                        l.Push(lua.LString(info.head))
                                }
                                if e := l.PCall(1, 0, nil); e !=nil { 
                                        log.Println(e.Error())
                                }
                        }
                        return 0
                case <-sessionQuit:
                        return 0
		}
	}
        return 0
}
func ServerUploadToRemote(requestid, sessionid int, sessionQuit chan bool, ma *Machine, l *lua.LState) int {
        from := l.CheckString(1)
        to := l.CheckString(2)
        b, err := ioutil.ReadFile(from)
        if err != nil {
                l.Push(lua.LString(err.Error()))
        } else {
                l.Push(lua.LNil)
        }
        t := &pipe.FileCmd{requestid, to, b}
	c:=make(chan responseT)
	requestMgr.AddRequest(&requestT{id:requestid, m:ma,waitC:c, closeC:make(chan bool)})
	defer requestMgr.RemoveRequest(requestid)
	pipe.Send(ma.conn, pipe.UploadFile, t)
	for {
		select {
		case info:= <- c:
                        msg:=info.msg
                        if l.Get(3).Type() == lua.LTFunction {
                                l.Push(l.Get(3))
                                l.Push(lua.LString(msg))
                                if e := l.PCall(1, 0, nil); e !=nil {
                                        log.Println(e.Error())
                                }
                        }
                        return 0
                case <-sessionQuit:
                        return 0
		}
	}
        return 0
}

func SendToRemote(requestid, sessionid int, sessionQuit chan bool, ma *Machine, l *lua.LState) int {
	s := l.CheckString(1)
	sec := l.CheckInt(2)
	c:=make(chan responseT)
	requestMgr.AddRequest(&requestT{id:requestid, m:ma,waitC:c, closeC:make(chan bool)})
	k:=&pipe.RequestCmd{sessionid, uint(requestid), s}
	pipe.Send(ma.conn, pipe.Request, k)
	t:=time.NewTicker(time.Second*time.Duration(sec))
	defer requestMgr.RemoveRequest(requestid)
	for {
		select {
		case info:= <- c:
			switch info.head {
			case "recv":
				msg:=info.msg
				if l.Get(3).Type() == lua.LTFunction {
                                        l.Push(l.Get(3))
					l.Push(lua.LString(msg))
					if e := l.PCall(1, 0, nil); e !=nil {
						log.Println(e.Error())
					}
				}
			case "end":
				l.Push(lua.LString(info.msg))
				return 1
			}
		case <-t.C:
			return 0
                case <-sessionQuit:
                        return 0
		}
	}
        t.Stop()
        return 0
}
//-------------------------
func ClientReadCallBack(conn *websocket.Conn, head string, arg []byte) {
       //log.Println("read callback", head, len(arg))
       switch head {
       case "getgrouplist":
	       RemoteTbl.RLock()
	       b, _ := json.Marshal(RemoteTbl)
	       RemoteTbl.RUnlock()
	       WSWrite(conn, []byte("grouplist"), b)
	case "opengroup":
		groupName := string(arg)
                tbl := make(map[string]*buttonConfig)
		LuaActionTblLock.RLock()
		buttons, h := LuaActionTbl[groupName]
		LuaActionTblLock.RUnlock()
		if h {
                        for k, v := range buttons {
                                if !v.Hide {
                                        tbl[k] = v
                                }
                        }
			b,_:=json.Marshal(tbl)
			WSWrite(conn, []byte("buttonlist"), b)
		}
	case "click":
		go ClickAction(string(arg), conn, "", 0)
        case "cancel":
		ClientTbl.CancelAction(string(arg))
        case "input":
                ar := strings.SplitN(string(arg), ":", 2)
                if len(ar) != 2 {
                        return
                }
                id, _ := strconv.Atoi(ar[0])
                q:=requestMgr.GetRequest(int(id))
                if q == nil {
                        return
                }
                select {
                case <-q.closeC:
                case q.waitC <- responseT{ar[1], ""}:
                }
       }
}

func OnRecvMsg(s pipe.ResponseCmd) {
	id := s.Id
	q:=requestMgr.GetRequest(int(id))
	//log.Println("get request", id , q)
	if q == nil {
		return
	}
	select {
	case <-q.closeC:
	case q.waitC <- responseT{s.Action, s.Cmd}:
	}
}

package server
import "net/http"
import "html/template"
import "log"
import "time"
import "../pipe"
import "bufio"
import "sync"
import "encoding/binary"
import "errors"
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
        //a:=int(c[0])+int(c[1]>>8)
        //g:=int(c[2])+int(c[3]>>8)+int(c[4]>>16)+int(c[5]>>24)


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

func ClickAction(file string, conn *websocket.Conn) {
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
	ca:= func(ma *Machine) {
		action := group+":"+ma.Nick+":"+f
		ClientTbl.AddAction(action)
		defer ClientTbl.RemoveAction(action)
                session := client.AddSession()
		single := (ClientTbl.HasActionSession(action)<=1)
                defer client.DelSession(session.id)
		l := lua.NewState()
		l.OpenLibs()
		l.SetGlobal("MachineGroup", lua.LString(group))
		l.SetGlobal("MachineName", lua.LString(ma.Nick))
		l.SetGlobal("MachineAddr", lua.LString(ma.conn.RemoteAddr().String()))
                common.InitCommon(l, session.quitC)
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
	}
	if m == "all" {
		t := ms.GetAll()
		for _, machine := range t {
			go ca(machine)
		}
	} else {
		machine := ms.Get(m)
		if machine == nil {
			return
		}
		go ca(machine)
	}
}

//-------------------------
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
		buttons, h := LuaActionTbl[groupName]
		if h {
			b,_:=json.Marshal(buttons)
			WSWrite(conn, []byte("buttonlist"), b)
		}
	case "click":
		go ClickAction(string(arg), conn)

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

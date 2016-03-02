package server

import "flag"
import "log"
import "net"
import "net/url"
import "os"
import "golang.org/x/net/websocket"
import "sync"
import "github.com/yuin/gopher-lua"
import "path/filepath"

var service = flag.String("service", "127.0.0.1:8888", "for remote client connect")
var webservice = flag.String("web", "127.0.0.1:8080", "http server port")

type ClientTblT struct {
	tbl map[*websocket.Conn]bool
	sync.RWMutex
}
func (c *ClientTblT) Broadcast(head, b []byte) {
        log.Println("begin Broadcast");
        c.RLock()
        for c, _ :=range c.tbl {
                WSWrite(c, head, b)
        }
        c.RUnlock()
        log.Println("end Broadcast");
}
func (c *ClientTblT) Add(conn *websocket.Conn) {
        c.Lock()
        c.tbl[conn] = true
        c.Unlock()
}
func (c *ClientTblT) Del(conn *websocket.Conn) {
        c.Lock()
        delete(c.tbl, conn)
        c.Unlock()
}

type sessionInfo struct {
        quitC chan bool
        id int
}

type Machine struct {
        Group string
        Nick string
        conn net.Conn
        sessionTbl map[int]*sessionInfo
        sessionLock sync.RWMutex
}

func (m *Machine) Init() {
        m.sessionTbl = make(map[int]*sessionInfo)
}

func (m *Machine) AddSession() *sessionInfo {
        id := genId()
        s := &sessionInfo{make(chan bool), id}
        m.sessionLock.Lock()
        m.sessionTbl[id] = s
        m.sessionLock.Unlock()
        return s
}

func (m *Machine) DelSession(id int) {
        m.sessionLock.Lock()
        s, b := m.sessionTbl[id]
        if b {
                close(s.quitC)
                delete(m.sessionTbl, id)
        }
        m.sessionLock.Unlock()
}

func (m *Machine) OnRemove() {
        for id, _ := range m.sessionTbl {
                m.DelSession(id)
        }
}

type Machines struct {
        Name string
        Tbl map[string]*Machine
        sync.RWMutex
}
func (m *Machines) Add(nick string, conn net.Conn) {
        m.Lock()
        _m := &Machine{Group:m.Name, Nick:nick, conn:conn}
        _m.Init()
        m.Tbl[nick] = _m
        m.Unlock()
}
func (m *Machines) Empty() bool {
        m.RLock()
        empty := len(m.Tbl)
        m.RUnlock()
        return empty == 0
}
func (m *Machines) Get(nick string) *Machine {
        m.RLock()
        mm, _ := m.Tbl[nick]
        m.RUnlock()
        return mm
}
func (m *Machines) GetAll() []*Machine {
	t:=[]*Machine{}
        m.RLock()
	for _, mm:= range m.Tbl {
		t=append(t, mm)
	}
        m.RUnlock()
        return t
}
func (m *Machines) Del(nick string) {
        _m := m.Get(nick)
        if _m != nil {
                _m.OnRemove()
        }
        m.Lock()
        delete(m.Tbl, nick)
        m.Unlock()
}
type RemoteTblT struct {
        Tbl map[string]*Machines
        conntbl map[net.Conn]*Machine
        sync.RWMutex
}
func (c *RemoteTblT) Add(group, nick string, conn net.Conn) {
        c.Lock()
        g, have := c.Tbl[group]
        if !have {
                g = &Machines{Tbl:make(map[string]*Machine), Name:group}
                c.Tbl[group] = g
        }
        g.Add(nick, conn)
        m:=g.Get(nick)
        c.conntbl[conn]  = m
        c.Unlock()
}
func (c *RemoteTblT) GetMachines(g string) *Machines {
        c.RLock()
        m, _ := c.Tbl[g]
        c.RUnlock()
	return m
}
func (c *RemoteTblT) Get(conn net.Conn) *Machine {
        c.RLock()
        m, _ := c.conntbl[conn]
        c.RUnlock()
	return m
}
func (c *RemoteTblT) Del(conn net.Conn) {
        c.Lock()
        m, have := c.conntbl[conn]
        if have {
                g, _have := c.Tbl[m.Group]
                if _have {
                        g.Del(m.Nick)
                }
		if g.Empty() {
			delete(c.Tbl, m.Group)
		}
                delete(c.conntbl, conn)
        }
        c.Unlock()
}

var ClientTbl *ClientTblT
var RemoteTbl *RemoteTblT
type buttonConfig struct {
        Name string
}
var LuaActionTbl map[string](map[string]*buttonConfig)
func Init() {
	ClientTbl = &ClientTblT{}
	ClientTbl.tbl = make(map[*websocket.Conn]bool)
        RemoteTbl  = &RemoteTblT{Tbl:make(map[string]*Machines), conntbl:make(map[net.Conn]*Machine)}
	flag.Parse()
        er:=InitAdminPort(*webservice)
        if er !=nil {
                log.Println("web server fail", er.Error())
                return
	}
	ScanButtons()
	Listen(*service)
}

func ScanButtons() {
	LuaActionTbl = make(map[string](map[string]*buttonConfig))
	filepath.Walk("./logic", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if err != nil {
				log.Println("search button fail", err.Error())
			} else {
				if filepath.Dir(filepath.Dir(path)) == "logic" && filepath.Ext(path) == ".lua" && filepath.Base(filepath.Dir(path)) != "internal" {
					name := info.Name()
					l := len(name)
					groupName := filepath.Base(filepath.Dir(path))
					buttonName := string([]byte(name)[:l-4])
					g, h:=LuaActionTbl[groupName]
					if !h {
						g = make(map[string]*buttonConfig)
						LuaActionTbl[groupName] = g
					}
                                        config := &buttonConfig{Name:buttonName}
                                        g[buttonName] = config
					log.Println("find button:", groupName, buttonName)
                                        ls := lua.NewState()
                                        ls.SetGlobal("bInit", lua.LBool(true))
                                        if err := ls.DoFile(path); err != nil {
                                                panic(err)
                                        }
                                        if t:= ls.Get(-1); t.Type()==lua.LTTable {
                                                v := t.(*lua.LTable).RawGetString("name")
                                                if v.Type() == lua.LTString {
                                                        s, _:= v.(lua.LString)
                                                        config.Name = url.QueryEscape(string(s))
                                                        log.Println("name", s)
                                                }
                                        }
				}
			}
		}
		return nil
	})
}
/*l := lua.NewState()
lua.OpenLibraries(l)
//RegLuaFunc(l, "add_button", add_button)
if err := lua.DoFile(l, "logic/lua/init.lua"); err != nil {
	panic(err)
}
//callButtonAction(l, "qa2test1")
*/
/*func RegLuaFunc(l *lua.State, name string, f func(l *lua.State) int) {
	l.PushGoFunction(f)
	l.SetGlobal(name)
}

func callButtonAction(l *lua.State, groupNick string) {
	l.Field(lua.RegistryIndex, groupNick)
	if l.IsFunction(-1) {
		l.Call(0, 0)
	} else {
		log.Println("call button action fail:", groupNick)
	}
}

func add_button(l *lua.State) int {
	n := l.Top()
	if n == 3 {
		groupName, _ := l.ToString(1)
		nick, _ := l.ToString(2)

		l.SetField(lua.RegistryIndex, groupName+nick)
		log.Println("add button for", groupName, nick)
	} else {
		log.Println("warning, not enough arg for add_button")
	}
	return 0
}*/

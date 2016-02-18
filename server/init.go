package server

import "flag"
import "log"
import "net"
import "os"
import "golang.org/x/net/websocket"
import "sync"
//import "github.com/Shopify/go-lua"
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

type Machine struct {
        Group string
        Nick string
        conn net.Conn
}
type Machines struct {
        Name string
        Tbl map[string]*Machine
        sync.RWMutex
}
func (m *Machines) Add(nick string, conn net.Conn) {
        m.Lock()
        m.Tbl[nick] = &Machine{m.Name, nick, conn}
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
var LuaActionTbl map[string](map[string]bool)
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
	LuaActionTbl = make(map[string](map[string]bool))
	filepath.Walk("./logic", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if err != nil {
				log.Println("search button fail", err.Error())
			} else {
				if filepath.Dir(filepath.Dir(path)) == "logic" && filepath.Ext(path) == ".lua" {
					name := info.Name()
					l := len(name)
					groupName := filepath.Base(filepath.Dir(path))
					buttonName := string([]byte(name)[:l-4])
					g, h:=LuaActionTbl[groupName]
					if !h {
						g = make(map[string]bool)
						LuaActionTbl[groupName] = g
					}
					g[buttonName] = true
					log.Println("find button:", groupName, buttonName)
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

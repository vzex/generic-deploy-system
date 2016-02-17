package server

import "flag"
import "log"
import "net"
import "golang.org/x/net/websocket"
import "sync"

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
	Listen(*service)
}

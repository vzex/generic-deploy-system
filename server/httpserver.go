package server
import "net/http"
import "html/template"
import "log"
import "bufio"
import "encoding/binary"
import "errors"
import "strings"
import "golang.org/x/net/websocket"
import "encoding/json"
import "github.com/Shopify/go-lua"

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
        return nil
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
	ar := strings.Split(file, ":")
	if len(ar) != 2 {return}
	f, m := ar[0], ar[1]
	_ar := strings.Split(f, "/")
	if len(_ar) != 2 {return}
	group := _ar[0]
	ms := RemoteTbl.GetMachines(group)
	log.Println("test", f, m, group)
	if ms ==nil {
		return
	}
	ca:= func(ma *Machine) {
		l := lua.NewState()
		lua.OpenLibraries(l)
		l.PushString(ma.Nick)
		l.SetGlobal("MachineName")
		l.PushString(ma.conn.RemoteAddr().String())
		l.SetGlobal("MachineAddr")
		if err := lua.DoFile(l, "logic/"+ f + ".lua"); err != nil {
			WSWrite(conn, []byte("warning"), []byte(err.Error()))
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

func ClientReadCallBack(conn *websocket.Conn, head string, arg []byte) {
       log.Println("read callback", head, len(arg))
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


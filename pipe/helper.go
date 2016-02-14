package pipe

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"strconv"
	"time"
	"net/http"
	"encoding/json"
)

type Cmd uint16

var handler codec.MsgpackHandle

func Encode(cmd Cmd, info interface{}) []byte {
	var buf bytes.Buffer
	var buf2 bytes.Buffer
	enc := codec.NewEncoder(&buf2, &handler)
	enc.Encode(info)
	b := buf2.Bytes()

	binary.Write(&buf, binary.LittleEndian, uint32(len(b)))
	binary.Write(&buf, binary.LittleEndian, uint16(cmd))
	binary.Write(&buf, binary.LittleEndian, b)
	return buf.Bytes()
}

func PackLua(cmd Cmd, data string) []byte {
	return Pack(cmd, []byte(data))
}
func Pack(cmd Cmd, data []byte) []byte {
	var buf bytes.Buffer
	b := data

	binary.Write(&buf, binary.LittleEndian, uint32(len(b)))
	binary.Write(&buf, binary.LittleEndian, uint16(cmd))
	binary.Write(&buf, binary.LittleEndian, b)
	return buf.Bytes()
}

func Decode(str []byte) (Cmd, []byte) {
	l := len(str)
	if l < 4 {
		return Invalid, []byte{}
	}
	buf := bytes.NewReader(str)
	var l1 uint32
	binary.Read(buf, binary.LittleEndian, &l1)
	needL := l1 + 2 + 4
	if uint32(l) < needL {
		return Invalid, []byte{}
	}
	var cmd uint16
	binary.Read(buf, binary.LittleEndian, &cmd)
	b := make([]byte, l1)
	copy(b, str[6:needL])
	return Cmd(cmd), b
}

func DecodeBytes(b []byte, info interface{}) {
	dec := codec.NewDecoderBytes(b, &handler)
	dec.Decode(&info)
}

func Send(conn net.Conn, cmd Cmd, info interface{}) (int, error) {
	if conn == nil {
		return 0, nil
	}
	b := Encode(cmd, info)
        n, err := conn.Write(b)
        if err != nil {
                log.Println("send error", conn.RemoteAddr(), cmd)
                conn.Close()
        }
        return n, err
}

type HelperInfo struct {
	Conn  net.Conn
	Cmd   Cmd
	Bytes []byte
}

type routeInfo struct {
        tbl map[Cmd]routeMode
}

type routeMode struct {
        dest *net.Conn
	inform bool
}

func (r *routeInfo) Add(cmd Cmd, conn *net.Conn, inform bool) {
        r.tbl[cmd] = routeMode{dest:conn, inform: inform}
}
func NewRouteInfo() *routeInfo {
        return &routeInfo{tbl:make(map[Cmd]routeMode)}
}

func NewInnerServer(addr string, c chan *HelperInfo, realPort *int) bool {
        return NewInnerServerWithRoute(addr, c, NewRouteInfo(), realPort)
}

func NewInnerServerWithRoute(addr string, c chan *HelperInfo, route *routeInfo, realPort *int) bool {
	return NewInnerServerWithRouteAndQuit(addr, c, route, make(chan bool), realPort)
}
func NewInnerServerWithRouteAndQuit(addr string, c chan *HelperInfo, route *routeInfo, q chan bool, realPort *int) bool {
	server, err := Listen(addr)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	if realPort != nil {
		*realPort = server.conn.Addr().(*net.TCPAddr).Port
	}
	in, out := make(chan net.Conn), make(chan net.Conn)
	callback := make(chan MsgCallback)
        quit := make(chan bool)
	go func() {
		msgtbl := make(map[net.Conn]string)
                _out:
		for {
			select {
                        case <-quit:
                                break _out
			case conn := <-in:
				//log.Println("conn in", conn.RemoteAddr().String())
				c <- &HelperInfo{conn, Enter, []byte{}}
				msgtbl[conn] = ""
			case conn := <-out:
				//log.Println("conn out", conn.RemoteAddr().String())
				c <- &HelperInfo{conn, Leave, []byte{}}
				delete(msgtbl, conn)
			case info := <-callback:
				s, _ := msgtbl[info.Conn]
				s += string(info.Msg)
                                //log.Println("conn recv", len(s))
				msgtbl[info.Conn] = s
				for {
					cmd, bytes := Decode([]byte(s))
					l := len(bytes)
                                        if cmd == Invalid {
						break
					}
					consumeL := l + 6
					s = string([]byte(s)[consumeL:])
					msgtbl[info.Conn] = s
                                        _info, have := route.tbl[cmd]

                                        if have && *_info.dest != nil {
                                                (*_info.dest).Write(Pack(cmd, bytes))
						if !_info.inform {
							continue
						}
                                        } else if have {
                                                log.Println("no conn", cmd)
                                        }
					c <- &HelperInfo{info.Conn, cmd, bytes}
				}
			}
		}
	}()
	log.Println("begin serve", addr)
	server.Serve(in, out, callback, q)
        close(quit)
        c <- &HelperInfo{nil, Shutdown, []byte{}}
	return true
}


type InnerClient struct {
	conn net.Conn
	tbl map[byte]func()
}

func NewInnerClient(addr string, c chan *HelperInfo) *InnerClient {
	client := &InnerClient{}
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	go func() {
		msg := ""
		arr := make([]byte, 1000)
		reader := bufio.NewReader(conn)
		for {
			size, err := reader.Read(arr)
			if err != nil {
				break
			}
			msg += string(arr[:size])
			for {
				cmd, bytes := Decode([]byte(msg))
				if cmd == Invalid {
					break
				}
				l := len(bytes)
				consumeL := l + 6
				msg = string([]byte(msg)[consumeL:])
				c <- &HelperInfo{conn, cmd, bytes}
			}
		}
		c <- &HelperInfo{conn, Shutdown, []byte{}}
	}()
	client.conn = conn
	client.tbl = make(map[byte]func())
	return client
}

func NewAdminHanderTbl() map[string]cmdHandler  {
        return make(map[string]cmdHandler)
}

type cmdHandler func(w http.ResponseWriter, r *http.Request) (result string, bSuccess bool)
type handlerResult struct {
        Code int
        Msg  string
}

func InitAdminPort(port int, adminTbl map[string]cmdHandler) error {
        mux := http.NewServeMux()
        f := func(w http.ResponseWriter, r *http.Request) {
                adminHandler(w, r, adminTbl)
        }
        mux.HandleFunc("/admin", f)
        addr := "0.0.0.0:" + strconv.Itoa(port)
        server := &http.Server{Addr: addr, Handler: mux}
        listener, err := net.Listen("tcp", addr)
        if err != nil {
                return err
        }
        log.Println("begin serve admin port:", addr)
        go server.Serve(listener)
        return nil
}

func adminHandler(w http.ResponseWriter, r *http.Request, adminTbl map[string]cmdHandler) {
        command := r.FormValue("cmd")
        if command != "" {
                handler, bHave := adminTbl[command]
                if bHave {
                        result, bOk := handler(w, r)
                        if bOk {
                                res, _ := json.Marshal(handlerResult{Code: 200, Msg: result})
                                w.Write([]byte(res))
                        } else {
                                res, _ := json.Marshal(handlerResult{Code: 201, Msg: result})
                                w.Write([]byte(res))
                        }
                        return
                }
        }
        res, _ := json.Marshal(handlerResult{Code: 202, Msg: "invalid command"})
        w.Write([]byte(res))
}

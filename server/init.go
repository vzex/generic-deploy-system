package server

import "flag"
import "log"
import "golang.org/x/net/websocket"
import "sync"

var service = flag.String("service", "127.0.0.1:8888", "for remote client connect")
var webservice = flag.String("web", "127.0.0.1:8080", "http server port")
type ClientTblT struct {
	tbl map[*websocket.Conn]bool
	sync.RWMutex
}
var ClientTbl *ClientTblT
func Init() {
	ClientTbl = &ClientTblT{}
	ClientTbl.tbl = make(map[*websocket.Conn]bool)
	flag.Parse()
        er:=InitAdminPort(*webservice)
        if er !=nil {
                log.Println("web server fail", er.Error())
                return
        }
	Listen(*service)
}

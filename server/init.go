package server

import "flag"
import "log"

var service = flag.String("service", "", "for remote client connect")
var webservice = flag.String("web", ":8080", "http server port")
var websocket = flag.String("websocket", "10.240.160.17:8081", "http web-socket port")
func Init() {
	flag.Parse()
        er:=InitAdminPort(*webservice, *websocket)
        if er !=nil {
                log.Println("web server fail", er.Error())
                return
        }
	Listen(*service)
}

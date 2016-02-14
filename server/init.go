package server

import "flag"

var service = flag.String("service", "", "for remote client connect")
var webservice = flag.String("web", ":8080", "http server port")
var websocket = flag.String("websocket", ":8081", "http web-socket port")
func Init() {
	flag.Parse()
	Listen(*service)
}

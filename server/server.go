package server

import "../pipe"
import "log"
func Listen(addr string) {
	c := make(chan *pipe.HelperInfo)
	go func() {
		for {
			select {
			case info := <-c:
				switch info.Cmd {
				case pipe.Leave:
                                        log.Println("leave", info.Conn.RemoteAddr().String())
				case pipe.Enter:
                                        log.Println("enter", info.Conn.RemoteAddr().String())
                                case pipe.Request:
                                        var s string
                                        pipe.DecodeBytes(info.Bytes, &s)
                                        log.Println("request", s)
                                case pipe.Response:
                                        log.Println("response")
				}
			}
		}
	}()
	pipe.NewInnerServer(addr, c, nil)
}

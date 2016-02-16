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
					for c, _ :=range ClientTbl.tbl {
						c.Write([]byte("leave:"+info.Conn.RemoteAddr().String()))
					}
				case pipe.Enter:
                                        log.Println("enter", info.Conn.RemoteAddr().String())
                                case pipe.Request:
                                        var s string
                                        pipe.DecodeBytes(info.Bytes, &s)
                                        log.Println("request", s)
                                case pipe.RegRemote:
                                        var _info pipe.RemoteInfo
                                        pipe.DecodeBytes(info.Bytes, &_info)
                                        log.Println("request", _info.Group, _info.Nick, info.Conn.RemoteAddr().String())
					for c, _ :=range ClientTbl.tbl {
						c.Write([]byte("enter:"+_info.Group+":"+_info.Nick))
					}
                                case pipe.Response:
                                        log.Println("response")
				}
			}
		}
	}()
	pipe.NewInnerServer(addr, c, nil)
}

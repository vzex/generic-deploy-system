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
                                        ClientTbl.Broadcast([]byte("leave"), []byte(info.Conn.RemoteAddr().String()))
				case pipe.Enter:
                                        log.Println("enter", info.Conn.RemoteAddr().String())
                                case pipe.Request:
                                        var s string
                                        pipe.DecodeBytes(info.Bytes, &s)
                                        log.Println("request", s)
                                case pipe.RegRemote:
                                        var _info pipe.RemoteInfo
                                        pipe.DecodeBytes(info.Bytes, &_info)
                                        log.Println("request begin ", _info.Group, _info.Nick, info.Conn.RemoteAddr().String())
                                        RemoteTbl.Add(_info.Group, _info.Nick, info.Conn)
                                        log.Println("end request");
                                        ClientTbl.Broadcast([]byte("enter"), []byte(_info.Group+":"+_info.Nick))
                                case pipe.Response:
                                        log.Println("response")
				}
			}
		}
	}()
	pipe.NewInnerServer(addr, c, nil)
}

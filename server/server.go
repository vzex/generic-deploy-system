package server

import "../pipe"
import "log"
import "encoding/json"

func Listen(addr string) {
	c := make(chan *pipe.HelperInfo)
	go func() {
		for {
			select {
			case info := <-c:
				switch info.Cmd {
				case pipe.Leave:
                                        log.Println("leave", info.Conn.RemoteAddr().String())
					m := RemoteTbl.Get(info.Conn)
					RemoteTbl.Del(info.Conn)
					if m!=nil {
						b, _ := json.Marshal(m)
						ClientTbl.Broadcast([]byte("leave"), b)
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
                                        log.Println("request begin ", _info.Group, _info.Nick, info.Conn.RemoteAddr().String())
                                        RemoteTbl.Add(_info.Group, _info.Nick, info.Conn)
                                        log.Println("end request");
					m := RemoteTbl.Get(info.Conn)
					if m!=nil {
						b, _ := json.Marshal(m)
						ClientTbl.Broadcast([]byte("enter"), b)
					}
                                case pipe.Response:
					var s pipe.ResponseCmd
					pipe.DecodeBytes(info.Bytes, &s)
					OnRecvMsg(s)
				}
			}
		}
	}()
	pipe.NewInnerServer(addr, c, nil)
}

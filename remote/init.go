package remote
import "flag"
import "log"
import "../pipe"

var service = flag.String("service", "127.0.0.1:8888", "")
var groupName = flag.String("group", "", "")
var nickName = flag.String("nick", "", "")
func Init() {
	flag.Parse()
	println("remote init")
        c:=make(chan *pipe.HelperInfo)
	d:=make(chan bool)
        go func() {
                for {
                        select {
                        case info:=<-c:
                                switch info.Cmd {
				case pipe.Shutdown:
					d<-true
                                case pipe.Request:
                                        log.Println("request")
                                case pipe.Response:
                                        log.Println("response")
                                }
                        }
                }
        }()
        client := pipe.NewInnerClient(*service, c)
        if client!=nil {
		info := &pipe.RemoteInfo{}
		info.Group = *groupName
		info.Nick = *nickName
               client.Send(pipe.RegRemote, info)
	       <-d
        } else {
		log.Println("dial fail")
	}
}

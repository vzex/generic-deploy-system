package remote
import "flag"
import "log"
import "../pipe"

var service = flag.String("service", "", "")
var groupName = flag.String("group", "", "")
var nickName = flag.String("nick", "", "")
func Init() {
	flag.Parse()
	println("remote init")
        c:=make(chan *pipe.HelperInfo)
        go func() {
                for {
                        select {
                        case info:=<-c:
                                switch info.Cmd {
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
               client.Send(pipe.Request, "haha")
        }
}

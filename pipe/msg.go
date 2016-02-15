package pipe
import "time"

const (
	//server spec cmd
	Invalid   Cmd = iota
	Shutdown //自身服务关闭
	Enter   //远程连入
	Leave   //远程连出
        Request
        Response
)

type Action byte

type RequestCmd struct {
	Id uint64
	Cmd string
	OverTime time.Time
}

package pipe

const (
	//server spec cmd
	Invalid   Cmd = iota
	Shutdown //自身服务关闭
	Enter   //远程连入
	Leave   //远程连出
)

type Action byte

//server common cmd
type CommonErrMsgCmd struct {
	Id uint64
	AccountName string
	Msg string
	Action Action
}

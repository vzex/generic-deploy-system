package pipe

const (
	//server spec cmd
	Invalid   Cmd = iota
	Shutdown //自身服务关闭
	Enter   //远程连入
	Leave   //远程连出
        Request
        Response
	RegRemote
)

type RemoteInfo struct {
	Group string
	Nick string
}

type RequestCmd struct {
	Id uint
	Cmd string
}

type ResponseCmd struct {
	Id uint
	Cmd string
	Action string
}

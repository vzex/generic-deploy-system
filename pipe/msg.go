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
        CancelRequest
        UploadFile
        DownloadFile
)

type RemoteInfo struct {
	Group string
	Nick string
}

type RequestCmd struct {
        SessionId int
	Id uint
	Cmd string
}

type ResponseCmd struct {
	Id uint
	Cmd string
	Action string
}

type FileCmd struct {
        Id int
        Name string
        Data []byte
}

type ResponseUploadFileCmd struct {
        Id int
        Name string
        Data []byte
}

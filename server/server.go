package server

import "../pipe"
func Listen(addr string) {
	c := make(chan *pipe.HelperInfo)
	go func() {
		for {
			select {
			case info := <-c:
				switch info.Cmd {
				case pipe.Leave:
				case pipe.Enter:
				}
			}
		}
	}()
	pipe.NewInnerServer(addr, c, nil)
}

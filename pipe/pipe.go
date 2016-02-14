package pipe

import (
	"bufio"
	"log"
	"net"
)

type MsgCallback struct {
	Conn net.Conn
	Msg  []byte
}

type Server struct {
	conn    net.Listener      //server listener
	clients map[net.Conn]bool //clients
	quit    chan bool
	in      chan net.Conn

	callbackIn  chan net.Conn
	callbackOut chan net.Conn
	callbackMsg chan MsgCallback
}

func Listen(addr string) (*Server, error) {
	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	} else {
		return &Server{conn: conn, clients: make(map[net.Conn]bool), quit: make(chan bool), in: make(chan net.Conn)}, nil
	}
}

func (s *Server) msgloop(quit chan bool) {
out:
	for {
		select {
		case <- quit:
                        go func() {close(s.quit)} ()
		case <-s.quit:
			for conn := range s.clients {
				s.callbackOut <- conn
				conn.Close()
			}
			s.clients = make(map[net.Conn]bool)
			s.conn.Close()
			break out
		case conn := <-s.in:
			s.clients[conn] = true
			s.callbackIn <- conn
			go func() {
				arr := make([]byte, 1000)
				reader := bufio.NewReader(conn)
				for {
					size, err := reader.Read(arr)
					if err != nil {
						break
					}
                                        d := make([]byte, size)
                                        copy(d, arr[:size])
					s.callbackMsg <- MsgCallback{conn, d}
				}
				s.callbackOut <- conn
				delete(s.clients, conn)
			}()
		}
	}
}

func (s *Server) Serve(in, out chan net.Conn, msgCallback chan MsgCallback, quit chan bool) error {
	s.callbackIn = in
	s.callbackOut = out
	s.callbackMsg = msgCallback
	var _err error
	go s.msgloop(quit)
	for {
		conn, err := s.conn.Accept()
		if err != nil {
			_err = err
			log.Println("server loop quit", err.Error())
			break
		}
		select {
		case <-s.quit:
		case s.in <- conn:
		}
	}
	return _err
}

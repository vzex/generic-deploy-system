package server
import "net/http"
import "net"
import "log"
import "strconv"

func InitAdminPort(port int) error {
        mux := http.NewServeMux()
        f := func(w http.ResponseWriter, r *http.Request) {
        }
        mux.HandleFunc("/admin", f)
        addr := "0.0.0.0:" + strconv.Itoa(port)
        server := &http.Server{Addr: addr, Handler: mux}
        listener, err := net.Listen("tcp", addr)
        if err != nil {
                return err 
        }   
        log.Println("begin serve admin port:", addr)
        go server.Serve(listener)
        return nil 
}



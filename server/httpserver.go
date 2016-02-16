package server
import "net/http"
import "html/template"
import "log"
import "io"
import "golang.org/x/net/websocket"

func InitAdminPort(addr string) error {
        http.Handle("/css/", http.FileServer(http.Dir("website")))
        http.Handle("/fonts/", http.FileServer(http.Dir("website")))
        http.Handle("/js/", http.FileServer(http.Dir("website")))
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                t, er := template.ParseFiles("website/index.html")
                if er == nil {
                        t.Execute(w, addr)
                } else {
                        log.Println("err:", er.Error())
                }
        })
	http.Handle("/ws", websocket.Handler(ProcessClient))
        log.Println("begin serve http:", addr)
        go http.ListenAndServe(addr, nil)
        return nil
}

func ProcessClient(ws *websocket.Conn) {
	log.Println("begin echo")
	ClientTbl.Lock()
	ClientTbl.tbl[ws] = true
	ClientTbl.Unlock()
	io.Copy(ws, ws)
	log.Println("end echo")
	ClientTbl.Lock()
	delete(ClientTbl.tbl, ws)
	ClientTbl.Unlock()
}

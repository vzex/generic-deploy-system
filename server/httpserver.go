package server
import "net/http"
import "html/template"
import "log"

func InitAdminPort(addr, websock string) error {
        http.Handle("/css/", http.FileServer(http.Dir("website")))
        http.Handle("/fonts/", http.FileServer(http.Dir("website")))
        http.Handle("/js/", http.FileServer(http.Dir("website")))
        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                t, er := template.ParseFiles("website/index.html")
                if er == nil {
                        t.Execute(w, websock)
                } else {
                        log.Println("err:", er.Error())
                }
        })
        log.Println("begin serve http:", addr)
        er := http.ListenAndServe(addr, nil)
        return er
}



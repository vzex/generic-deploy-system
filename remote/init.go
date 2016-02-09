package remote
import "flag"

var service = flag.String("service", "", "")
var groupName = flag.String("group", "", "")
var nickName = flag.String("nick", "", "")
func Init() {
	flag.Parse()
	println("remote init")
}

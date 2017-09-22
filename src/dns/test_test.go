package dns

import "testing"
//import "strconv"
import "time"
import (
	"fmt"
	//"net/rpc"
	//"net"
	//"log"
	"net/http"
	//"raft"
	//"dns"

)

func TestWeb(t *testing.T) {
	StartDNS()
	time.Sleep(300 * time.Second)
}

func TestProxy(t *testing.T) {
	//Serve("8080")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
	time.Sleep(100 * time.Second)

}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("here:%v", r)
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}



package raftkv

import "testing"
//import "strconv"
import (
	"time"
	"raft"
)

func TestWeb(t *testing.T){

	me := 0
	StartKVServer(me, raft.MakePersister(), -1)
	time.Sleep(150*time.Second)
}

package raftkv

import (
	"encoding/gob"
	"log"
	"raft"
	"sync"
	"time"
	"sort"
	//"html/template"
	"net/http"
	"io/ioutil"
	"regexp"
	"fmt"
	"net/rpc"
	"net"
	"strconv"
)

const Debug = 1

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

type Op struct {
			// Your definitions here.
			// Field names must start with capital letters,
			// otherwise RPC will break.
	Key      string
	Password string
	Kind     string //post, login
	Value    Content
	Id       string
	ReqId    int
}

type RaftKV struct {
	mu             sync.Mutex
	me             int
	rf             *raft.Raft
	applyCh        chan raft.ApplyMsg

	maxraftstate   int // snapshot if log grows this big

			   // Your definitions here.
	postDatabase   map[int]Content
	userCurrentLog map[string]int
	logAck         map[int]chan Op

	user           map[string]string
	alreadyLogin   map[string]string
}

func (kv *RaftKV) AppendEntryToLog(entry Op) bool {
	index, _, isLeader := kv.rf.Start(entry)
	if !isLeader {
		return false
	}

	kv.mu.Lock()
	ch, ok := kv.logAck[index]
	if !ok {
		ch = make(chan Op, 1)
		kv.logAck[index] = ch
	}
	kv.mu.Unlock()
	select {
	case op := <-ch:
		return op == entry
	case <-time.After(1000 * time.Millisecond):
	//log.Printf("timeout\n")
		return false
	}
}

/*func (kv *RaftKV) CheckDup(id string, reqid int) bool {
	//kv.mu.Lock()
	//defer kv.mu.Unlock()
	v, ok := kv.userCurrentLog[id]
	if ok {
		return v >= reqid
	}
	return false
}*/

func (kv *RaftKV) DNSGetLeader(args GetLeaderArgs, reply *GetLeaderReply) error {
	reply.IsLeader = kv.rf.IsLeader()
	return nil
}

func (kv *RaftKV) DNSLogIn(args LoginArgs, reply *DnsReply) error {
	reply.IsLogin = kv.loginForm(args)
	if reply.IsLogin {
		reply.Data.UserName = args.UserName
	}

	reply.Data.Post = kv.postDatabase
	return nil
}

func (kv *RaftKV) DNSLogout(args LogoutArgs, reply *DnsReply) error {
	reply.IsLogin = kv.logoutForm(args)
	if reply.IsLogin {

	}
	//if reply.IsLogin {
	reply.Data.Post = kv.postDatabase
	//}
	return nil
}

func (kv *RaftKV) DNSPost(args PostArgs, reply *DnsReply) error {
	reply.IsLogin = true
	kv.postForm(args)
	reply.Data.UserName = kv.alreadyLogin[args.UserIP]
	reply.Data.Post = kv.postDatabase
	return nil
}

func (kv *RaftKV) DNSHomePage(args MainPageArgs, reply *DnsReply) error {
	//fmt.Printf("receive dns request\n")

	fmt.Printf("request HomePage\n")
	reply.IsLogin, reply.Data = kv.homePage(args)
	fmt.Printf("reply.Data is %v\n", reply.IsLogin)

	return nil

}
func (kv *RaftKV) DNSLike(args LikeArgs, reply *DnsReply) error {
	reply.IsLogin = true
	kv.likeForm(args)
	reply.Data.UserName = kv.alreadyLogin[args.UserIP]
	reply.Data.Post = kv.postDatabase
	return nil
}

func (kv *RaftKV) likeForm(r LikeArgs) {
	var reply PutAppendReply
	key := strconv.Itoa(r.PostID)
	args := PutAppendArgs{
		Id:r.UserIP,
		Op:"Like",
		ReqID:kv.userCurrentLog[kv.alreadyLogin[r.UserIP]] + 1,
		Key:key,
	}
	kv.like(&args, &reply)
}

func (kv *RaftKV) like(args *PutAppendArgs, reply *PutAppendReply) {
	entry := Op{Kind:args.Op, Key:args.Key, Id:args.Id, ReqId:args.ReqID}
	ok := kv.AppendEntryToLog(entry)
	if !ok {
		reply.WrongLeader = true
	} else {
		reply.WrongLeader = false
		reply.Err = OK
	}
}

func (kv *RaftKV) Post(args *PutAppendArgs, reply *PutAppendReply) {
	entry := Op{Kind:args.Op, Key:args.Key, Value:args.Value, Id:args.Id, ReqId:args.ReqID}
	ok := kv.AppendEntryToLog(entry)
	if !ok {
		reply.WrongLeader = true
	} else {
		reply.WrongLeader = false
		reply.Err = OK
	}
}

func (kv *RaftKV) login(args *loginArgs, reply *PutAppendReply) {
	entry := Op{Kind:args.Op, Key: args.Key, Password:args.Password, Id:args.Id, ReqId:args.ReqID}
	ok := kv.AppendEntryToLog(entry)
	if !ok {
		reply.WrongLeader = true
	} else {
		reply.WrongLeader = false
		reply.Err = OK

	}
}

func (kv *RaftKV) logout(args *logoutArgs, reply *PutAppendReply) {
	entry := Op{Kind:args.Op, Key: args.Key, Id:args.Id, ReqId:args.ReqID}
	ok := kv.AppendEntryToLog(entry)
	if !ok {
		reply.WrongLeader = true
	} else {
		reply.WrongLeader = false
		reply.Err = OK

	}
}

func (kv *RaftKV) Apply(args Op, index int) {
	switch args.Kind {
	case "Post":
		kv.postDatabase[index] = args.Value
	case "Login":
		kv.alreadyLogin[args.Id] = args.Key
	case "Logout":
		delete(kv.alreadyLogin, args.Id)
	case "Like":
		key, err := strconv.Atoi(args.Key)
		if err != nil {
			println(err)
		}
		title := kv.postDatabase[key].Title
		link := kv.postDatabase[key].Link
		likeNum := kv.postDatabase[key].LikeNum + 1
		kv.postDatabase[key] = Content{Title:title, Link:link, LikeNum:likeNum}
		fmt.Println(kv.postDatabase[key])

	}
	kv.userCurrentLog[kv.alreadyLogin[args.Id]] = args.ReqId
}



//
// the tester calls Kill() when a RaftKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *RaftKV) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
}

func Serve(kv *RaftKV, port string) {
	serv := rpc.NewServer()
	serv.Register(kv)
	oldMux := http.DefaultServeMux
	mux := http.NewServeMux()
	http.DefaultServeMux = mux
	serv.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	http.DefaultServeMux = oldMux
	l, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	go http.Serve(l, mux)
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots with persister.SaveSnapshot(),
// and Raft should save its state (including log) with persister.SaveRaftState().
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(me int, persister *raft.Persister, maxraftstate int) *RaftKV {
	// call gob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	gob.Register(Op{})
	gob.Register(MainPage{})
	kv := new(RaftKV)
	Serve(kv, ":8080")
	//StartKVServer(arith,servers, me, raft.MakePersister(), -1)


	kv.me = me
	kv.maxraftstate = maxraftstate

	// Your initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(me, persister, kv.applyCh)

	time.After(10 * time.Second)

	kv.postDatabase = make(map[int]Content)
	kv.userCurrentLog = make(map[string]int)
	kv.logAck = make(map[int]chan Op)
	kv.user = make(map[string]string)
	kv.alreadyLogin = make(map[string]string)

	//initially, store one user in our user database, username is abc, pwd is 123
	kv.user["abc"] = "123"

	go func() {
		for {
			msg := <-kv.applyCh

			op := msg.Command.(Op)
			kv.mu.Lock()
			kv.Apply(op, msg.Index)

			ch, ok := kv.logAck[msg.Index]
			if ok {
				select {
				case <-kv.logAck[msg.Index]:
				default:
				}
				ch <- op
			} else {
				kv.logAck[msg.Index] = make(chan Op, 1)
			}

			kv.mu.Unlock()

		}
	}()

	return kv
}

type Page struct {
	Title string
	Body  []byte
}

func (kv *RaftKV) loadPage(title string) (*Page, error) {
	var body []byte
	var newstring string
	var keys[] int

	for k := range kv.postDatabase {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, value := range keys {
		newstring += (kv.postDatabase[value].Title + " " + kv.postDatabase[value].Link + "\n")
	}
	body = []byte(newstring)
	return &Page{Title: title, Body: body}, nil
}

func (kv *RaftKV) makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	validPath := regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
	return func(w http.ResponseWriter, r *http.Request) {

		basicInfo := r.Method + " " + r.RequestURI + " " + r.Proto
		DPrintf(basicInfo)

		DPrintf("Detected IP address is : ", r.RemoteAddr)
		DPrintf("Real IP address could be : ", r.Header.Get("X-Forwarded-For"))

		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func (kv *RaftKV) getUserName(userIP string) (bool, string) {
	val, ok := kv.alreadyLogin[userIP]
	if ok {
		return true, val
	} else {
		return false, ""
	}
}

func (kv *RaftKV) homePage(r MainPageArgs) (bool, MainPage) {
	var display MainPage
	var IsLogin bool
	fmt.Println("open home page")
	IsLogin, display.UserName = kv.getUserName(r.UserIP)

	kv.postDatabase[0] = Content{Title:"google", Link:"www.google.com", LikeNum:0}
	display.Post = kv.postDatabase

	fmt.Printf("display is %v\n", display.Post)
	return IsLogin, display
}

func (kv *RaftKV) postForm(r PostArgs) {
	_, ok := kv.alreadyLogin[r.UserIP]
	if ok {
		DPrintf("log in already")
	}

	fmt.Println("username:", r.UserIP)

	var reply PutAppendReply
	args := PutAppendArgs{
		Id:r.UserIP,
		Op:"Post",
		ReqID:kv.userCurrentLog[kv.alreadyLogin[r.UserIP]] + 1,
		Key:kv.alreadyLogin[r.UserIP],
		Value:Content{Title:r.Title, Link:r.Link, LikeNum:0},
	}
	kv.Post(&args, &reply)

	//return isLogin
}

func (kv *RaftKV) loginForm(r LoginArgs) bool {
	var isLogin bool
	//var display MainPage
	_, ok := kv.alreadyLogin[r.UserIP]
	if ok {
		DPrintf("log in already")
	}

	fmt.Println("username:", r.UserName)
	fmt.Println("password:", r.Password)

	value, ok := kv.user[r.UserName]
	if !ok {
		fmt.Println("not in database")
		isLogin = false
	} else {
		if (value != r.Password) {
			fmt.Println("not valid")
			isLogin = false
		} else {
			var reply PutAppendReply
			args := loginArgs{Password:r.Password, ReqID:0, Op:"Login", Key: r.UserName, Id:r.UserIP}
			kv.login(&args, &reply)
			isLogin = true
		}

	}
	return isLogin
}

func (kv *RaftKV) logoutForm(r LogoutArgs) bool {
	var isLogin bool
	_, ok := kv.alreadyLogin[r.UserIP]
	if ok {
		//DPrintf("log in already")
	}

	var reply PutAppendReply
	args := logoutArgs{ReqID:kv.userCurrentLog[kv.alreadyLogin[r.UserIP]] + 1, Op:"Logout", Id:r.UserIP}
	kv.logout(&args, &reply)

	isLogin = false

	return isLogin

}

func (kv *RaftKV) loadPageLocal(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

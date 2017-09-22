package dns

import (
	"net/http"
	"net/rpc"
	"fmt"
	"encoding/gob"
	"bytes"
	"log"
	"html/template"
	"strconv"
)

var leaderID int
var ip []string
var count int
var servers []*rpc.Client

func StartDNS() {
	gob.Register(LoginArgs{})
	gob.Register(DnsReply{})
	gob.Register(MainPage{})
	gob.Register(GetLeaderArgs{})
	gob.Register(GetLeaderReply{})

	leaderID = 0
	ip = make([]string, 3)
	ip[1] = "155.41.107.179"
	ip[0] = "168.122.9.173"
	ip[2] = "168.122.204.53"


	count = 3

	servers = make([]*rpc.Client, 3)

	for i := 0; i < count; i++ {
		client, err := rpc.DialHTTP("tcp", ip[i] + ":8080")
		//err := 'a'
		for err != nil {
			log.Fatal("dialing:", err)
			fmt.Println("connecting")
			client, err = rpc.DialHTTP("tcp", ip[i] + ":8080")
		}
		servers[i] = client
		fmt.Println("connected %d", i)
	}
	fmt.Printf("connection success\n")

	go func() {
		for {
			listenToPort()
		}
	}()

}

func dial() {
	for i := 0; i < count; i++ {
		client, err := rpc.DialHTTP("tcp", ip[i] + ":8080")
		//err := 'a'
		if err != nil {
			client, err = rpc.DialHTTP("tcp", ip[i] + ":8080")
		}
		servers[i] = client
		fmt.Println("connected %d", i)
	}
}

func findLeader(servers []*rpc.Client, userIP string) {
	dial()
	fmt.Println("receive find leader request")

	for i := 0; i < count; i++ {
		var reply GetLeaderReply
		var network bytes.Buffer
		encoder := gob.NewEncoder(&network)
		p := &GetLeaderArgs{}
		encoder.Encode(p)
		if servers[i] != nil {
			//fmt.Println("server[i] is ",servers[i])
			err := servers[i].Call("RaftKV.DNSGetLeader", p, &reply)
			if err != nil {
				//log.Fatal("arith error:",err)
				fmt.Println("there is error findLeader", err, "try to call:", i)
			}
			if reply.IsLeader {
				fmt.Println("Do we find a leader?:", reply.IsLeader)
				fmt.Println("leader is :", i)
				leaderID = i
				return
			}
		}
	}

}

func listenToPort() {
	//isLeader := kv.rf.IsLeader()
	//DPrintf("%v isLeader:%v", kv.me, isLeader)


	http.DefaultServeMux = new(http.ServeMux)
	mux := http.NewServeMux()
	mux.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("../html/"))))
	mux.HandleFunc("/homePage", homePage)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/post", post)
	mux.HandleFunc("/logout", logout)
	mux.HandleFunc("/like", like)
	http.ListenAndServe(":8000", mux)

}

func homePage(w http.ResponseWriter, r *http.Request) {
	userIP := r.RemoteAddr
	var reply DnsReply
	findLeader(servers, userIP)

	args := MainPageArgs{Op:"HomePage", UserIP:userIP}
	err := servers[leaderID].Call("RaftKV.DNSHomePage", &args, &reply)
	fmt.Println(" error", err)
	for err != nil {
		findLeader(servers, userIP)
		err = servers[leaderID].Call("RaftKV.DNSHomePage", &args, &reply)
		//log.Fatal("arith error:",err)
		fmt.Println("there is error homepage,", err, "id:", leaderID)
	}
	fmt.Println("open home page", reply.Data)

	fmt.Println("data is", reply.Data)
	db := reply.Data

	t, error := template.ParseFiles("../html/homepage.html")
	if error != nil {
		fmt.Printf("error : %v\n", error)
		return
	}

	t.Execute(w, db)

}

func login(w http.ResponseWriter, r *http.Request) {
	userIP := r.RemoteAddr
	var reply DnsReply
	fmt.Println("receive login request")
	findLeader(servers, userIP)

	basicInfo := r.Method + " " + r.RequestURI + " " + r.Proto

	fmt.Println(basicInfo)

	if r.Method == "GET" {
		t, _ := template.ParseFiles("../html/login.html")
		t.Execute(w, nil)
	} else {
		r.ParseForm()
		username := r.Form["username"][0]
		password := r.Form["password"][0]

		args := LoginArgs{UserIP:userIP, UserName: username, Password:password}
		err := servers[leaderID].Call("RaftKV.DNSLogIn", &args, &reply)
		//fmt.Println(" error",err)
		for err != nil {
			findLeader(servers, userIP)
			err = servers[leaderID].Call("RaftKV.DNSLogIn", &args, &reply)
			//log.Fatal("arith error:",err)
			fmt.Println("there is error login", err, "id :", leaderID)
		}

		//fmt.Println("login",reply.Data)

		db := reply.Data
		t, error := template.ParseFiles("../html/homepage.html")
		if error != nil {
			fmt.Printf("error : %v\n", error)
			return
		}
		//fmt.Print("parsefile\n")
		t.Execute(w, db)

	}
}

func post(w http.ResponseWriter, r *http.Request) {
	userIP := r.RemoteAddr
	var reply DnsReply
	//findLeader(servers,userIP)

	fmt.Println("receive post request")

	basicInfo := r.Method + " " + r.RequestURI + " " + r.Proto

	fmt.Println(basicInfo)

	if r.Method == "GET" {
		t, _ := template.ParseFiles("../html/post.html")
		t.Execute(w, nil)
	} else {
		r.ParseForm()
		title := r.Form["title"][0]
		link := r.Form["link"][0]

		args := PostArgs{UserIP:userIP, Link: link, Title:title}
		findLeader(servers, userIP)
		fmt.Println(leaderID)
		err := servers[leaderID].Call("RaftKV.DNSPost", &args, &reply)
		fmt.Println(" error oh no", err)
		for err != nil {
			fmt.Println("it is here")
			findLeader(servers, userIP)
			err = servers[leaderID].Call("RaftKV.DNSPost", &args, &reply)
			//log.Fatal("arith error:",err)
			fmt.Println("there is error post", err)
		}

		fmt.Println("post", reply.Data)

		db := reply.Data
		t, error := template.ParseFiles("../html/homepage.html")
		if error != nil {
			//fmt.Printf("error : %v\n",error)
			return
		}
		//fmt.Print("parsefile\n")
		t.Execute(w, db)

	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	userIP := r.RemoteAddr
	var reply DnsReply
	findLeader(servers, userIP)

	//basicInfo := r.Method + " " + r.RequestURI + " " + r.Proto

	//fmt.Println(basicInfo)
	fmt.Println("receive logout request")
	//password :=r.Form["password"][0]


	args := LogoutArgs{UserIP:userIP}
	err := servers[leaderID].Call("RaftKV.DNSLogout", &args, &reply)
	//fmt.Println(" error",err)
	for err != nil {
		findLeader(servers, userIP)
		err = servers[leaderID].Call("RaftKV.DNSLogout", &args, &reply)
		//log.Fatal("arith error:",err)
		fmt.Println("there is error logout", err, "id :", leaderID)
	}

	//fmt.Println("logout",reply.Data)

	db := reply.Data
	t, error := template.ParseFiles("../html/homepage.html")
	if error != nil {
		fmt.Printf("error : %v\n", error)
		return
	}
	//fmt.Print("parsefile\n")
	t.Execute(w, db)

}

func like(w http.ResponseWriter, r *http.Request) {

	userIP := r.RemoteAddr
	var reply DnsReply
	//findLeader(servers,userIP)

	fmt.Println("receive like request")

	basicInfo := r.Method + " " + r.RequestURI + " " + r.Proto

	fmt.Println(basicInfo)

	r.ParseForm()
	postID := r.Form["postId"][0]

	i, _ := strconv.Atoi(postID)

	fmt.Println(postID)

	args := LikeArgs{PostID: i, UserIP:userIP}
	findLeader(servers, userIP)
	fmt.Println(leaderID)
	err := servers[leaderID].Call("RaftKV.DNSLike", &args, &reply)
	//fmt.Println(" error oh no",err)
	for err != nil {
		//fmt.Println("it is here")
		findLeader(servers, userIP)
		err = servers[leaderID].Call("RaftKV.DNSLike", &args, &reply)
		//log.Fatal("arith error:",err)
		//fmt.Println("there is error like",err)
	}

	//fmt.Println("Like",reply.Data)

	db := reply.Data
	t, error := template.ParseFiles("../html/homepage.html")
	if error != nil {
		//fmt.Printf("error : %v\n",error)
		return
	}
	//fmt.Print("parsefile\n")
	t.Execute(w, db)

}
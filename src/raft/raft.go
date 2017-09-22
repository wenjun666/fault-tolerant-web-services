package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import "sync"
import (
	"time"
	"math/rand"
	//"debug/elf"
	"bytes"
	"encoding/gob"
	//"math"
	//"fmt"
	//"fmt"
	"net/rpc"
	"fmt"
	"net/http"
	"net"
)

// import "bytes"
// import "encoding/gob"



//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make().
//
type ApplyMsg struct {
	Index       int
	Command     interface{}
	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*rpc.Client       // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	// 2A
	currentTerm 	int
	votedFor	int
	log		[]*Log
	commitIndex	int
	lastApplied	int
	//nextIndex	[]int
	//matchIndex	[]int
	leader		bool
	voteCount	int
	voteNum		int
	hearNum		int
	//state           int// 0 leader 1 candidate 2 follower

	followChan	chan bool
	leaderChan	chan bool
	candidateChan	chan bool
	commitChan	chan bool

	nextIndex     []int
	matchIndex    []int

}

type Log struct{
	Command		interface{}
	Term 		int
	Index 		int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm, rf.leader
}

func (rf *Raft) IsLeader() bool {
	return rf.leader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	 w := new(bytes.Buffer)
	 e := gob.NewEncoder(w)
	 e.Encode(rf.currentTerm)
	 e.Encode(rf.votedFor)
	 e.Encode(rf.log)
	 data := w.Bytes()
	 rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	// Your code here (2C).
	// Example:
	r := bytes.NewBuffer(data)
	d := gob.NewDecoder(r)
	d.Decode(&rf.currentTerm)
	d.Decode(&rf.votedFor)
	d.Decode(&rf.log)
	//if data == nil || len(data) < 1 { // bootstrap without any state?
	//	return
	//}
}

//2A
type AppendEntriesArgs struct{
	Term 		int
	LeaderId	int
	PrevLogIndex	int
	PrevLogTerm	int
	Entries 	[]*Log
	LeaderCommit	int
}

type AppendEntriesReply struct{
	Term 		int
	Success 	bool
	MatchIndex	int
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term		int	//candidate's term
	CandidateID	int	//candidate requesting vote
	LastLogIndex	int	//index of candidate's last log entry
	LastLogTerm	int	//term of candidate's last log entry
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term 		int	//currentTerm, for candidate to update itself
	VoteGranted  	bool	//tre means candidate received vote
}


func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	reply.Success = false

	if (args.Term < rf.currentTerm) {
		reply.Term = rf.currentTerm
		return nil
	}else{
		//during the append entry, if leadercommit > commitIndex, update
		if (args.Term > rf.currentTerm) {
			rf.leader = false
			rf.votedFor = -1
			rf.currentTerm = args.Term
		}
		reply.Term = rf.currentTerm

		//if not have prevLogIndex, or preLogTerm not match return false
		if (args.PrevLogIndex > len(rf.log) - 1) {
			reply.MatchIndex = len(rf.log)-1
			go func(){
				rf.followChan <- true
			}()
			return nil
		}else{
			lastMatch := args.PrevLogIndex
			matchLoop:
			for i := args.PrevLogIndex + 1; i < len(rf.log) && i < args.PrevLogIndex + 1 + len(args.Entries) ; i++ {
				if rf.log[i].Term != args.Entries[i-(args.PrevLogIndex + 1)].Term {
					break matchLoop
				}
				lastMatch = i
			}
			//if not all match, remove log entries after the matchIndex
			if lastMatch != len(rf.log) - 1 {
				rf.log = rf.log[:lastMatch+1]
			}

			//add new log
			reply.MatchIndex = lastMatch
			for i := lastMatch - args.PrevLogIndex; i < len(args.Entries); i++ {
				rf.log = append(rf.log,args.Entries[i])
				reply.MatchIndex = len(rf.log)-1
			}
		}

		//commit new log
		if (args.LeaderCommit > rf.commitIndex) {
			if (args.LeaderCommit > len(rf.log)-1) {
				rf.commitIndex = len(rf.log)-1
			}else{
				rf.commitIndex = args.LeaderCommit

			}
			go func(){
				rf.commitChan <- true
			}()
		}

		reply.Success = true

		//reset follower timer
		go func(){
			rf.followChan <- true
		}()
		//fmt.Printf("server %d reset timer\n",rf.me)
	}
	return nil
	//rf.me = reply.Term

}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	if ok != nil {
		fmt.Printf("can't connect to server %s\n",server)
		fmt.Println(ok)
		ok2 := rf.reconnectServer(server)
		if ok2 {
			ok = rf.peers[server].Call("Raft.AppendEntries", args, reply)
		}
	}
	return ok
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	
	follower := false
	reply.VoteGranted = false
	
	// if args.term is lower, won't vote for it
	if (args.Term < rf.currentTerm) {
		//if term is smaller than currentTerm
		//reply.VoteGranted = false
		reply.Term = rf.currentTerm
		return nil
	}

	CandLogTerm := args.LastLogTerm
	CandLogIndex := args.LastLogIndex
	candidate := args.CandidateID

	lastLogTerm := rf.log[ len(rf.log) - 1].Term
	lastLogIndex := len(rf.log) - 1


	if (args.Term > rf.currentTerm) {
		follower = true
		rf.currentTerm = args.Term
		rf.votedFor = -1
	}
	reply.Term = rf.currentTerm
	
	if (follower) {
		rf.leader = false;
	}

	if (rf.votedFor == -1 || rf.votedFor == candidate) && (lastLogTerm < CandLogTerm || (lastLogTerm == CandLogTerm && lastLogIndex <= CandLogIndex)) {
		rf.votedFor = candidate
		reply.VoteGranted = true
	}
	if (reply.VoteGranted) {
		go func(){
			rf.followChan <- true
		}()
		return nil
	}
	return nil
	
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) error {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	if ok != nil {
		fmt.Printf("can't connect to server %s\n",server)
		fmt.Println(ok)
		ok2 := rf.reconnectServer(server)
		if ok2 {
			ok = rf.peers[server].Call("Raft.RequestVote", args, reply)
		}
	}
	return ok
}


//if a server is down, try to reconnect it
func (rf *Raft) reconnectServer(server int) bool{
	ok := false
	//count := 3
	ip := make([]string, 3)
	// hard code ip address of three server
	ip[1] = "xxx.xx.xxx.xxx"
	ip[0] = "xxx.xxx.xxx.xxx"
	ip[2] = "xxx.xxx.xxx.xxx"

	fmt.Println("Hi, try to connect to server",server)
	client, err := rpc.DialHTTP("tcp", ip[server] + ":8000")
	//err := 'a'
	if err != nil {
		//log.Fatal("dialing:", err)
		fmt.Println("error")
		client, err = rpc.DialHTTP("tcp", ip[server] + ":8000")
	} else {
		leader :=rf.IsLeader()
		if leader {
			rf.mu.Lock()
			rf.peers[server] = client
			rf.matchIndex[server] = 0
			rf.nextIndex[server] = 1
			rf.mu.Unlock()
		}
		ok = true
		fmt.Println("connected server", server)
	}
	return ok
}
//custimize new rpc tunnel (online solution)
func Serve(rf *Raft, port string) {
	serv := rpc.NewServer()
	serv.Register(rf)
	oldMux := http.DefaultServeMux
	mux := http.NewServeMux()
	http.DefaultServeMux = mux

	serv.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	http.DefaultServeMux = oldMux

	l, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	go func() {

		http.Serve(l, mux)

	}()
}

func getPeers(rf *Raft, me int, n int) []*rpc.Client {
	//rpc.Register(rf)

	Serve(rf, ":8000")
	count := n
	ip := make([]string, n)

	ip[1] = "xxx.xxx.xxx.xxx"
	ip[0] = "xxx.xxx.xxx.xxx"
	ip[2] = "xxx.xxx.xxx.xxx"

	servers := make([]*rpc.Client, n)
	for i := 0; i < count; i++ {
		if i != me {
			var client *rpc.Client
			client, err := rpc.DialHTTP("tcp", ip[i] + ":8000")
			//err := 'a'
			for err != nil {
				//log.Fatal("dialing:", err)
				client, err = rpc.DialHTTP("tcp", ip[i] + ":8000")
			}
			servers[i] = client
			fmt.Println("connected %d", i)
		}
	}
	fmt.Printf("connected success\n")
	return servers
}



//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	index := -1
	term := -1
	isLeader := rf.leader
	if isLeader {
		term = rf.currentTerm
		newLog := &Log{
			Command: command,
			Term: term,
			Index: len(rf.log),
		}
		
		index = len(rf.log)
		rf.log = append(rf.log, newLog)
		rf.persist()
	}
	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(me int, persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = getPeers(rf, me, 3)
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	//2A
	rf.currentTerm = 0 // term of each server starts from 0
	rf.votedFor = -1  // initially not vote for any server
	rf.log = append(rf.log, &Log{nil,0,0} ) // initially set
	rf.commitIndex = 0	//initialized to 0
	rf.lastApplied = 0	//initialized to 0
	rf.leader = false
	rf.followChan = make(chan bool)
	rf.candidateChan = make(chan bool)
	rf.leaderChan = make(chan bool)
	rf.commitChan = make(chan bool)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	go Follower(rf)
	go Apply(rf,applyCh)
	return rf
}

func Follower(rf *Raft) {
	rf.mu.Lock()
	rf.voteNum = 0
	rf.voteCount = 0
	rf.hearNum = 0
	rf.leader = false
	rf.mu.Unlock()
	loop:
	for {
		t := time.Duration(rand.Intn(300) + 550)
		timer := time.NewTicker(t * time.Millisecond)
		select {
		case <-timer.C:
			go Candidate(rf)
			break loop
		case <-rf.followChan:

		}
	}
}

func (rf *Raft)leaderStateInitial(){
	for i := range rf.nextIndex {
		rf.nextIndex[i] = 1
		rf.matchIndex[i] = 0
	}
}

func (rf *Raft)updateCommit() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	N := rf.commitIndex
	for i:=rf.commitIndex + 1; i < len(rf.log);i++ {
		majority := 1
		for j := range rf.peers {
			if j != rf.me && rf.matchIndex[j] >= i && rf.log[i].Term == rf.currentTerm {
				majority++
			}
		}
		if majority > len(rf.peers)/2 {
			N = i
		}
	}
	if N!= rf.commitIndex {
		rf.commitIndex = N
		rf.commitChan <- true
	}
}

func Leader(rf *Raft) {
	
	// initial the nextIndex and matchIndex
	rf.mu.Lock()
	rf.leader = true
	rf.voteCount = 0
	rf.voteNum = 0
	rf.mu.Unlock()
	rf.nextIndex = make([]int, 50)
	rf.matchIndex = make([]int, 50)

	rf.leaderStateInitial()
	
	leaderl:
	for {
		rf.mu.Lock()
		rf.hearNum = 0
		isleader := rf.leader
		beforeTerm := rf.currentTerm
		rf.mu.Unlock()

		if (!isleader) {
			rf.mu.Lock()
			rf.votedFor = -1
			rf.hearNum = 0
			rf.mu.Unlock()
			go Follower(rf)
			break leaderl
		}
		//check if need to increase commitIndex
		rf.updateCommit()
		
		//follower := false

		//hearNum used to count how many server has reply
		rpcloop:
		for i := range rf.peers {
			rf.mu.Lock()
			if (rf.leader == false) {
				//follower = true
				rf.mu.Unlock()
				break rpcloop
			}
			rf.mu.Unlock()
			if ( i != rf.me ) {
				rf.mu.Lock()
				args := &AppendEntriesArgs{
					Term: rf.currentTerm,
					LeaderId: rf.me,
					LeaderCommit:rf.commitIndex,
					PrevLogIndex: rf.matchIndex[i],
					PrevLogTerm: rf.log[rf.matchIndex[i]].Term,
					Entries:rf.log[rf.matchIndex[i]+1:],
				}
				rf.mu.Unlock()
				go func(i int,args *AppendEntriesArgs) {
					reply := &AppendEntriesReply{}
					ok := rf.sendAppendEntries(i,args,reply)
					if ok == nil {
						rf.mu.Lock()
						if (rf.leader){
							rf.hearNum++
						}
						rf.mu.Unlock()
						if (reply.Term > rf.currentTerm ) {
							rf.mu.Lock()
							rf.currentTerm = reply.Term
							rf.votedFor = -1
							rf.leader = false
							rf.mu.Unlock()
							rf.persist()
							//follower = true
							return
						}
						
						if reply.Success {
							rf.mu.Lock()
							rf.nextIndex[i] = reply.MatchIndex + 1
							rf.matchIndex[i] = reply.MatchIndex
							rf.mu.Unlock()
						}

					}
				}(i,args)
			}
		}
		time.Sleep(150*time.Millisecond)
		rf.mu.Lock()
		currentTerm := rf.currentTerm
		leader  := rf.leader
		rf.mu.Unlock()
		if (currentTerm != beforeTerm || !leader) {
			go func() {
				rf.followChan <- true
			}()
			//return
		}else {
			go func() {
				rf.leaderChan <- true
			}()
		}
		//fmt.Printf("leader %d finished all the heartbeat this round \n",rf.me)
		select{
		case <-rf.followChan:
			//fmt.Printf("leader %d receive higher term, become follower in term %d\n",rf.me,rf.currentTerm)
			rf.mu.Lock()
			rf.votedFor = -1
			rf.hearNum = 0
			rf.mu.Unlock()
			go Follower(rf)
			break leaderl
		case <-rf.leaderChan:
			//time.Sleep(150*time.Millisecond)
		}
	}


}

func Candidate(rf *Raft) {
	rf.mu.Lock()
	rf.currentTerm++ // at beginning of the election, increase the term by 1
	rf.votedFor = rf.me // vote for itself
	rf.voteCount = 0
	rf.voteNum = 0
	rf.leader = false
	rf.mu.Unlock()
	rf.persist()
	//t:= time.Duration(rand.Int63() % 300 + 510)
	t := time.Duration(rand.Intn(300) + 300)
	timerChan := time.NewTicker(t*time.Millisecond).C
	go Election(rf)
	select {
		case <-timerChan:
			go Candidate(rf)
		case <-rf.leaderChan:	// receive vote from majority of servers
			go Leader(rf)
		case <-rf.followChan:	// discover leader or new term
			go Follower(rf)
		}
}



func Election(rf *Raft) {
	defer rf.persist()
	rf.mu.Lock()
	rf.voteCount = 1
	rf.voteNum = 1
	args := &RequestVoteArgs{
		Term: rf.currentTerm,
		CandidateID: rf.me,
		LastLogIndex: rf.log[len(rf.log) - 1].Index,
		LastLogTerm: rf.log[len(rf.log) - 1].Term,
	}
	argTerm := rf.currentTerm
	rf.mu.Unlock()

	//loop:
	for i := range rf.peers{
		if(i != rf.me) {
			go func(i int){
				
				reply := &RequestVoteReply{}
				ok := rf.sendRequestVote(i,args,reply)
				
				if ok == nil{
					rf.mu.Lock()
					if (!rf.leader){
						rf.voteNum++
					}
					rf.mu.Unlock()

					if reply.Term > args.Term {
						rf.mu.Lock()
						rf.currentTerm = reply.Term
						rf.votedFor = -1
						rf.mu.Unlock()
						rf.persist()
					}
					if reply.VoteGranted{
						rf.mu.Lock()
						rf.voteCount++
						rf.mu.Unlock()
					}
				}
			}(i)

		}
	}
	time.Sleep(15*time.Millisecond)
	//if there is no enough server in the newtork
	//just return and wait time out for election
	rf.mu.Lock()
	if (rf.voteNum <= len(rf.peers)/2) {
		rf.voteNum = 0
		rf.voteCount = 0
		rf.mu.Unlock()

			return
	}
	rf.mu.Unlock()


	//check if term has changed
	//if term change, switch to follower an return
	rf.mu.Lock()
	curTerm := rf.currentTerm
	rf.mu.Unlock()
	if (argTerm < curTerm) {
		go func(){
			rf.followChan <- true
		}()
		return
	}

	//if majority of the server vote for candidate
	//become leader
	rf.mu.Lock()
	count := rf.voteCount
	rf.mu.Unlock()
	if (count <= len(rf.peers)/2) {
		go func(){
			rf.followChan <- true
		}()
	}else if (count > len(rf.peers)/2){
		//fmt.Printf("candidate %d win %d vote out of %d in term %d\n",rf.me,count,len(rf.peers),rf.currentTerm)
		go func(){
			rf.leaderChan <- true
		}()
	}
}

func Apply(rf *Raft,applyCh chan ApplyMsg) {
	for {
		select{
		case <- rf.commitChan :
			rf.mu.Lock()
			commitIndex := rf.commitIndex
			for i:= rf.lastApplied + 1; i <= commitIndex; i++ {
				//fmt.Printf("server %d commit log %d\n",rf.me,i)
				applyMsg := ApplyMsg{
					Index: i,
					Command:rf.log[i].Command,
				}
				applyCh <- applyMsg
				rf.lastApplied = i
				//fmt.Printf("server %d applyed command %v\n",rf.me,rf.log[i].Command)
			}
			rf.mu.Unlock()
		}
	}
}
package raftkv

const (
	OK = "OK"
	ErrNoKey = "ErrNoKey"
)

type Err string

// Put or Append
type PutAppendArgs struct {
		     // You'll have to add definitions here.
	Key   string
	Value Content
	Op    string // "Put" or "Append"
		     // You'll have to add definitions here.
		     // Field names must start with capital letters,
		     // otherwise RPC will break.
	Id    string
	ReqID int
}

type PutAppendReply struct {
	WrongLeader bool
	Err         Err
}

type GetArgs struct {
	Key   string
	// You'll have to add definitions here.
	Id    string
	ReqID int
}

type loginArgs struct {
	Password string
	ReqID    int
	Op       string
	Key      string
	Id       string
}

type GetReply struct {
	WrongLeader bool
	Err         Err
	Value       string
}

type MainPage struct{
	UserName string
	Post map[int]Content
}

type User struct {
	Username string
	Password string
}

type Content struct {
	Title string
	Link  string
	LikeNum	int
}

type GetLeaderArgs struct{
}

type GetLeaderReply struct{
	IsLeader bool
}

type MainPageArgs struct {
	Op string
	UserIP string
	//IsLeader bool
	// R *http.Request
	// W http.ResponseWriter
}


type LoginArgs struct{
	//IsLeader bool
	UserIP	 string
	UserName string
	Password string

}

type PostArgs struct{
	UserIP string
	Link string
	Title string
}

type LikeArgs struct{
	PostID int
	UserIP	string
}

type DnsReply struct {
	//IsLeader bool
	IsLogin	bool
	Data 	MainPage
}


type logoutArgs struct {
	ReqID    int
	Op       string
	Key      string
	Id       string
}


type LogoutArgs struct{
	UserIP string

}
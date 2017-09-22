package dns

type MainPage struct{
	UserName string
	Post map[int]Content
}


type Content struct {
	Title string
	Link  string
	LikeNum int
}


type MainPageArgs struct {
	 Op string
	 UserIP string
}


type LoginArgs struct{
	//IsLeader bool
	UserIP string
	UserName string
	Password string

}

type PostArgs struct{
	UserIP string
	Link string
	Title string
}


type DnsReply struct {
	IsLogin	bool
	Data 	MainPage
}

type GetLeaderArgs struct{

}

type GetLeaderReply struct{
	IsLeader bool
}


type LogoutArgs struct{
	UserIP string

}


type logoutArgs struct {
	ReqID    int
	Op       string
	Key      string
	Id       string
}



type LikeArgs struct{
	UserIP string
	PostID int

}
# Project Target
 To build a distributed, centralized, fault-tolerant website. There are multiple server running for the website. 
If one of the servers is down, the other servers will still make the website reachable to users. The function of the website is 
similar to Reddit, user can login, view links and post links to the website. To the client's’ point of view, the website makes no 
difference no matter which server they are connected to - if clients are logged in to a server which latter downs, 
the substitute leader should also show this status.

# instrcution
go to dns.go and raft.go, and specify the number of servers and their ip address.
```go
	ip = make([]string, 3)
	ip[1] = "xxx.xx.xxx.xxx"
	ip[0] = "xxx.xx.xxx.xxx"
	ip[2] = "xxx.xx.xxx.xxx"
```

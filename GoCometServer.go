package main

import (
	"container/list"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type User struct {
	uid            string
	session        string
	isOnline       bool
	GetMsgCnt      int
	ch             chan string
	MM             string
	LastAccessTime time.Time
	isVisitor      bool
}

//user = &User{uid: uid, session: "session", isOnline: false, GetMsgCnt: 0, ch: nil, MM: ""}
//user = &User{uid, "session", true, 0, nil, ""}

var UserDB map[string]*User //get obj from map[string]User will copy struct only
var ChatRoom map[string]*list.List

func Handler(w http.ResponseWriter, r *http.Request) {

	//	fmt.Fprintf(w, "<h1>welcome to go chat server %s!</h1>", r.URL.Path[1:])
	//	return

	File := r.URL.Path[1:]
	t, err := template.ParseFiles(File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
	return

}

func getmsg(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	callback := r.FormValue("callback")

	isVisitor := (uid == "")

	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "getmsg", Uid: uid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("GETMSG: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		if callback == "" {
			fmt.Fprintf(w, pub_tostring(rv))
		} else {
			fmt.Fprintf(w, callback+"(%s)", pub_tostring(rv))
		}
		return
	}

	//println("GETMSG: ", uid, tuser.GetMsgCnt)
	if user.GetMsgCnt > 0 {
		println("damn msgcnt")
		for {
			if user.GetMsgCnt <= 0 {
				break
			}
			println("GETMSG: Clean up ", uid)
			rv2 := returnValue2{IRet: 0, Cmd: "getmsg"}
			user.ch <- pub_tostring(rv2)
			runtime.Gosched()
			runtime.Gosched()
			//tuser.GetMsgCnt = tuser.GetMsgCnt - 1
		}
	}

	user.GetMsgCnt++
	defer func() {
		user.GetMsgCnt = user.GetMsgCnt - 1
	}()

	cookie := http.Cookie{Name: "session", Value: user.session, Path: "/", HttpOnly: true}
	http.SetCookie(w, &cookie)

	if isVisitor {
		cookie = http.Cookie{Name: "vid", Value: user.uid, Path: "/"}
		http.SetCookie(w, &cookie)
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10e9)
		timeout <- true
	}()

	select {
	case msg := <-user.ch:

		rv.IRet = 1
		rv.Msg = msg
		fmt.Fprintf(w, msg) //tuser.MM

		user.MM = ""
	case <-timeout:
		rv2 := returnValue2{IRet: 1, Cmd: "getmsg"}
		fmt.Fprintf(w, pub_tostring(rv2))

	}

}

// StringReplace -- replaces all occurences of rep with sub in src 
func StringReplace(src, rep, sub string) (n string) {
	// make sure the src has the char we want to replace. 
	if strings.Count(src, rep) > 0 {
		runes := src // convert to utf-8 runes. 
		for i := 0; i < len(runes); i++ {
			l := string(runes[i]) // grab our rune and convert back to string. 
			if l == rep {
				n += sub
			} else {
				n += l
			}
		}
		return n
	}
	return src
}
func say(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")
	msg := r.FormValue("msg")

	msg = StringReplace(msg, " ", "+")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "say", Uid: uid, Tuid: tuid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := UserDB[tuid]
	if ok {

	} else {
		tuser = &User{uid: tuid, session: "NA"}
		tuser.ch = make(chan string, 1000)
		UserDB[tuid] = tuser
		println("SETMSG: new to user ", tuid)
	}

	//msg = fmt.Sprintf("[msg]{act:say,uid:%s,tuid:%s,msg:%s,time:%s}", uid, tuid, msg, time.Now().String())
	//msg = fmt.Sprintf("[msg]%s say to %s:%s", uid, tuid, msg)
	tuser.MM = msg

	rv.IRet = 1
	rv.Msg = msg

	fmt.Fprintf(w, pub_tostring(rv))
	tuser.ch <- pub_tostring(rv)
}

func pub_tostring(rv interface{}) string {
	b, _ := json.Marshal(rv)
	fmt.Println(string(b))
	return string(b)
}

func logon(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	pass := r.FormValue("pass")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "logon", Uid: uid}

	if !pub_CheckPass(uid, pass) {
		fmt.Println("LOGIN FAIL: %s %s", uid, pass)
		time.Sleep(2e9)
		rv.Msg = "LOGIN FAIL"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	user.session, _ = pub_GenSession()
	user.isOnline = true

	cookie := http.Cookie{Name: "session", Value: user.session, Path: "/", HttpOnly: true}
	http.SetCookie(w, &cookie)

	//println("LOGIN SUCCESS: " + uid)

	rv.IRet = 1
	rv.Msg = "LOGIN SUCCESS"
	fmt.Fprintf(w, pub_tostring(rv))

}
func logoff(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "logoff", Uid: uid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	user.MM = "offline"
	user.session = "NA"
	user.isOnline = false

	rv.IRet = 1
	rv.Msg = "offline success"
	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)

	user.ch <- rvs

}

func joinroom(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	cid := r.FormValue("cid")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "joinroom", Uid: uid, Cid: cid}
	//var room *list.List

	user, ok := UserDB[uid]
	if ok {
	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("joinroom: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	room, ok := ChatRoom[cid]
	if ok {

	} else {
		room = list.New()
		ChatRoom[cid] = room
	}
	if !UserinList(room, user) {
		room.PushFront(user)
	}
	rv.IRet = 1
	rvs := pub_tostring(rv)

	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		value.ch <- rvs

	}
	runtime.Gosched()

	fmt.Fprintf(w, rvs)
}

func UserinList(room *list.List, user *User) bool {
	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		if user == value {
			return true
		}
	}
	return false
}
func DeleteUserinList(room *list.List, user *User) bool {
	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		if user == value {
			room.Remove(element)
			return true
		}
	}
	return false
}

func sayroom(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	cid := r.FormValue("cid")
	msg := r.FormValue("msg")

	msg = StringReplace(msg, " ", "+")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	println("[" + msg + "]")

	rv := returnValue{IRet: 0, Cmd: "sayroom", Uid: uid, Cid: cid}

	room, ok := ChatRoom[cid]
	if ok {

	} else {
		room = list.New()
		ChatRoom[cid] = room
	}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("joinroom: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	if !UserinList(room, user) {
		room.PushFront(user)
	}
	rv.IRet = 1
	rv.Msg = msg

	rvs := pub_tostring(rv)

	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		value.ch <- rvs

	}
	runtime.Gosched()
	fmt.Fprintf(w, rvs)
}

func leftroom(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	cid := r.FormValue("cid")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "leftroom", Uid: uid, Cid: cid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("leftroom: new user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	room, ok := ChatRoom[cid]
	if ok {

	} else {
		room = list.New()
		ChatRoom[cid] = room
	}

	DeleteUserinList(room, user)

	rv.IRet = 1
	rvs := pub_tostring(rv)

	for element := room.Front(); element != nil; element = element.Next() {
		value := element.Value.(*User)
		value.ch <- rvs
	}
	runtime.Gosched()

	if room.Len() <= 0 {
		delete(ChatRoom, cid)
		println("room deleted  ", cid)
	}

	fmt.Fprintf(w, rvs)
}

func call(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "call", Uid: uid, Tuid: tuid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := UserDB[tuid]
	if ok {

	} else {
		tuser = &User{uid: tuid, session: "NA"}
		tuser.ch = make(chan string, 1000)
		UserDB[tuid] = tuser
		println("SETMSG: new to user ", tuid)
	}

	rv.IRet = 1
	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)
	tuser.ch <- rvs
}

func cancelcall(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")
	if isVisitor {
		uid = pub_GetVid(r)
	}

	rv := returnValue{IRet: 0, Cmd: "cancelcall", Uid: uid, Tuid: tuid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := UserDB[tuid]
	if ok {

	} else {
		tuser = &User{uid: tuid, session: "NA"}
		tuser.ch = make(chan string, 1000)
		UserDB[tuid] = tuser
		println("SETMSG: new to user ", tuid)
	}

	rv.IRet = 1
	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)
	tuser.ch <- rvs

}

func acceptcall(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")

	//cid, _ := pub_GenSession()
	cid := uid + "_" + tuid

	rv := returnValue{IRet: 0, Cmd: "acceptcall", Uid: uid, Tuid: tuid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := UserDB[tuid]
	if ok {

	} else {

		tuser = &User{uid: tuid, session: "NA"}

		tuser.ch = make(chan string, 1000)
		UserDB[tuid] = tuser
		println("SETMSG: new to user ", tuid)
	}

	room, ok := ChatRoom[cid]
	if ok {

	} else {
		room = list.New()
		ChatRoom[cid] = room
	}

	rv.Cid = cid
	rv.IRet = 1

	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)
	tuser.ch <- rvs
}
func ignorecall(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "ignorecall", Uid: uid, Tuid: tuid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := UserDB[tuid]
	if ok {

	} else {
		tuser = &User{uid: tuid, session: "NA"}
		tuser.ch = make(chan string, 1000)
		UserDB[tuid] = tuser
		println("SETMSG: new to user ", tuid)
	}

	rv.IRet = 1
	rvs := pub_tostring(rv)

	fmt.Fprintf(w, rvs)
	tuser.ch <- rvs

}

func getuserstatus(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	tuid := r.FormValue("tuid")

	isVisitor := (uid == "")

	rv := returnValue{IRet: 0, Cmd: "getuserstatus", Uid: uid, Tuid: tuid}

	user, ok := UserDB[uid]
	if ok {

	} else {
		user = &User{uid: uid, session: "NA"}
		user.isVisitor = isVisitor
		user.ch = make(chan string, 1000)
		UserDB[uid] = user
		println("SETMSG: new from user ", uid)
	}

	if !pub_VerifySession(user, r) {
		time.Sleep(5e9)
		rv.IRet = 0
		rv.Msg = "Please Login"
		fmt.Fprintf(w, pub_tostring(rv))
		return
	}

	tuser, ok := UserDB[tuid]
	if ok {
		if tuser.isOnline {
			rv.IRet = 1
			rv.Msg = "Online"
			fmt.Fprintf(w, pub_tostring(rv))
		} else {
			rv.IRet = 1
			rv.Msg = "Offline"
			fmt.Fprintf(w, pub_tostring(rv))
		}

	} else {
		rv.IRet = 1
		rv.Msg = "Offline"
		fmt.Fprintf(w, pub_tostring(rv))
	}

}

var isBreak bool

func main() {
	UserDB = make(map[string]*User)
	ChatRoom = make(map[string]*list.List)

	readOptions()

	http.Handle("/html/", http.FileServer(http.Dir("")))

	http.HandleFunc("/", Handler)
	http.HandleFunc("/logon", logon)
	http.HandleFunc("/getmsg", getmsg)
	http.HandleFunc("/logoff", logoff)
	http.HandleFunc("/say", say)

	http.HandleFunc("/joinroom", joinroom)
	http.HandleFunc("/leftroom", leftroom)
	http.HandleFunc("/sayroom", sayroom)

	http.HandleFunc("/call", call)
	http.HandleFunc("/cancelcall", cancelcall)
	http.HandleFunc("/acceptcall", acceptcall)
	http.HandleFunc("/ignorecall", ignorecall)

	http.HandleFunc("/getuserstatus", getuserstatus)

	//http.ListenAndServe(":80", nil)
	isBreak = false
	go func() {
		for {
			time.Sleep(10e9)
			//fmt.Println("Checking timeout...")
			for _, user := range UserDB {
				//fmt.Println("Checking " + user.uid)
				if time.Since(user.LastAccessTime) > 60e9 {

					fmt.Println("dropping " + user.uid)
					delete(UserDB, user.uid)
					//UserDB[user.uid] = nil
				}

			}

			if isBreak {
				break
			}

		}
	}()

	http.ListenAndServe(ids.Listen, nil)

}

func pub_GenSession() (string, error) {
	uuid := make([]byte, 8)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	return hex.EncodeToString(uuid), nil
}

func pub_CheckPass(uid string, pass string) bool {
	result := false
	for _, id := range ids.Ids {
		if id.Id == uid {
			if pass == id.Password {
				result = true
			}
		}
	}
	return result
}
func pub_VerifySession(user *User, r *http.Request) bool {

	user.LastAccessTime = time.Now()

	if user.isVisitor {
		return true
	}

	//==========session ==========
	session := ""
	cookie, err := r.Cookie("session")
	if err == nil {
		session = cookie.Value
		//println(cookie.Value)
	}
	//==========session ==========

	if session != user.session {
		fmt.Println("pub_VerifySession %s %s/%s", user.uid, session, user.session)
		return false
	}

	return true

}

func pub_GetVid(r *http.Request) string {

	vid := ""

	//==========session ==========

	cookie, err := r.Cookie("vid")
	if err == nil {
		vid = cookie.Value
	}
	//==========session ==========

	if vid == "" {
		vid, _ = pub_GenSession()
	}

	return vid

}

//===========================
//  Read Options
//===========================

type idPassType struct {
	Id       string //captital is very import!!
	Password string
}

type idsType struct {
	Listen string
	Ids    []idPassType
}

type returnValue struct {
	IRet int
	Cmd  string
	Uid  string
	Tuid string
	Cid  string
	Msg  string
}

type returnValue2 struct {
	IRet int
	Cmd  string
}

var ids idsType

func readOptions() {
	file, e := ioutil.ReadFile("./httpserver.json")

	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(file))

	json.Unmarshal(file, &ids)
	fmt.Printf("Results: %v\n", ids)
}

//===========================

//	http.HandleFunc("/getmsg", getmsg) 
//        1.logon 2.say  3.logoff if in same dept
//         4.JoinRoom 5.leftroom 6.sayroom 
// 7.call 8.cancelcall )

//	http.HandleFunc("/logoff", logoff)
//	http.HandleFunc("/say", say)

//	http.HandleFunc("/joinroom", joinroom) 
//	http.HandleFunc("/leftroom", leftroom)
//	http.HandleFunc("/sayroom", sayroom)

//	http.HandleFunc("/call", call) //say
//	http.HandleFunc("/cancelcall", cancelcall) //say
//	http.HandleFunc("/acceptcall", acceptcall) //joinroom say
//	http.HandleFunc("/invite", invite) //joinroom say

//http.HandleFunc("/get", getEnv)
//
//func getEnv(writer http.ResponseWriter, req *http.Request) {
//	env := os.Environ()
//	writer.Write([]byte("<h1>Envirment</h1><br>"))
//	for _, v := range env {
//		writer.Write([]byte(v + "<br>"))
//	}
//	writer.Write([]byte("<br>"))
//}

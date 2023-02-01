package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Credential struct {
	DedeUserID int64
	sessdata   string
	bili_jct   string
	buvid3     string
}

func (c Credential) ToCookie() string {
	return fmt.Sprintf("DedeUserID=%v; SESSDATA=%v; bili_jct=%v; buvid3=%v", c.DedeUserID, c.sessdata, c.bili_jct, c.buvid3)
}

type Session struct {
	maxTs      int64
	maxSeqno   map[int64]int64
	exceptSelf bool
	credential Credential
}

type SessionList struct {
	TalkerID    int64 `json:"talker_id"`
	SessionType int64 `json:"session_type"`
	SessionTs   int64 `json:"session_ts"`
	MaxSeqno    int64 `json:"max_seqno"`
}

type Event struct {
	SenderUID    int64  `json:"sender_uid"`
	ReceiverType int64  `json:"receiver_type"`
	ReceiverID   int64  `json:"receiver_id"`
	MsgType      int64  `json:"msg_type"`
	Content      string `json:"content"`
	MsgSeqno     int64  `json:"msg_seqno"`
	Timestamp    int64  `json:"timestamp"`
	MsgKey       int64  `json:"msg_key"`
}

type ApiData struct {
	Data struct {
		Messages    []Event       `json:"messages"`
		SessionList []SessionList `json:"session_list"`
	} `json:"data"`
}

type TextContent struct {
	Content string `form:"content" json:"content"`
}

// 尝试解析文本消息
func (event Event) GetContent() string {
	var realContent TextContent
	err := json.Unmarshal([]byte(event.Content), &realContent)
	if printErr(err) {
		return realContent.Content
	}
	return ""
}

// 基础请求，自动添加请求头、Cookie
func (session Session) Request(method string, url string, data string) []byte {
	bodyReader := bytes.NewReader([]byte(data))
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bodyReader)
	if printErr(err) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Referer", "https://www.bilibili.com")
		req.Header.Add("User-Agent", "Mozilla/5.0")
		req.Header.Add("Cookie", session.credential.ToCookie())
		resp, err := client.Do(req)
		if printErr(err) {
			body, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			return body
		}
	}
	return []byte{}
}

// 获取指定用户的最近 30 条消息
func (session Session) FetchSessionMsgs(talkerID, sessionType, beginSeqno int64) {
	Url := "https://api.vc.bilibili.com/svr_sync/v1/svr_sync/fetch_session_msgs"
	Params := fmt.Sprintf("?talker_id=%v&session_type=%v&begin_seqno=%v", talkerID, sessionType, beginSeqno)

	var Api ApiData
	err := json.Unmarshal(session.Request("GET", Url+Params, ""), &Api)
	panicErr(err)

	for _, v := range Api.Data.Messages {
		if v.SenderUID != session.credential.DedeUserID || !session.exceptSelf {
			user := GetUserByUID(v.SenderUID)
			if user == nil {
				session.reply(v, "获取对象失败，请联系管理员。")
			}
			switch v.GetContent() {
			case "注册", "register", "token", "/注册", "/register", "/token":
				session.reply(v, user.Token)
			case "我", "个人信息", "信息", "等级":
				msg := fmt.Sprintf("权限：LV%v\n经验：%v\n监听列表：%v\n回传地址：%v密钥：%v", user.Level, user.XP, user.Watch, user.Url, user.Token)
				session.reply(v, msg)
			}
		}
	}
}

// 获取新接收的消息列表
func (session Session) NewSessions(beginTs int64) []SessionList {
	Url := "https://api.vc.bilibili.com/session_svr/v1/session_svr/new_sessions"
	Params := fmt.Sprintf("?begin_ts=%v&build=0&mobi_app=web", beginTs)

	var Api ApiData
	err := json.Unmarshal(session.Request("GET", Url+Params, ""), &Api)
	panicErr(err)

	return Api.Data.SessionList
}

// 获取已有消息列表
func (session Session) GetSessions(sessionType int64) []SessionList {
	Url := "https://api.vc.bilibili.com/session_svr/v1/session_svr/get_sessions"
	Params := fmt.Sprintf("?session_type=%v&group_fold=1&unfollow_fold=0&sort_rule=2&build=0&mobi_app=web", sessionType)

	var Api ApiData
	err := json.Unmarshal(session.Request("GET", Url+Params, ""), &Api)
	panicErr(err)

	return Api.Data.SessionList
}

// 发送消息
func (session Session) SendMsg(ReceiverID int64, content string) {
	Url := "https://api.vc.bilibili.com/web_im/v1/web_im/send_msg"
	Format := "msg[sender_uid]=%v&msg[receiver_id]=%v&msg[receiver_type]=1&msg[msg_type]=1&msg[msg_status]=0&msg[content]={\"content\":\"%v\"}&msg[dev_id]=B9A37BF3-AA9D-4076-A4D3-366AC8C4C5DB&msg[new_face_version]=0&msg[timestamp]=%v&from_filework=0&build=0&mobi_app=web&csrf=%v&csrf_token=%v"
	Data := fmt.Sprintf(Format, session.credential.DedeUserID, ReceiverID, content, time.Now().Unix(), session.credential.bili_jct, session.credential.bili_jct)

	session.Request("POST", Url, Data)
}

// 快速回复消息
func (session Session) reply(event Event, content string) {
	if session.credential.DedeUserID != event.SenderUID {
		session.SendMsg(event.SenderUID, content)
	}
}

// 间隔轮询
func (session Session) run(Interval int64) {
	// 初始化 只接收开始运行后的新消息
	for _, v := range session.GetSessions(4) {
		session.maxSeqno[v.TalkerID] = v.MaxSeqno
	}

	ticker := time.NewTicker(time.Duration(Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, v := range session.NewSessions(session.maxTs) {
			AnyTo(v.SessionTs > session.maxTs, &session.maxTs, v.SessionTs)
			go session.FetchSessionMsgs(v.TalkerID, v.SessionType, session.maxSeqno[v.TalkerID])
			session.maxSeqno[v.TalkerID] = v.MaxSeqno
		}
	}
}

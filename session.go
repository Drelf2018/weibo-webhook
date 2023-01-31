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

type GetSessionData struct {
	Data struct {
		SessionList []struct {
			TalkerID    int64 `json:"talker_id"`
			SessionType int64 `json:"session_type"`
			SessionTs   int64 `json:"session_ts"`
			MaxSeqno    int64 `json:"max_seqno"`
		} `json:"session_list"`
	} `json:"data"`
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

type TextContent struct {
	Content string `form:"content" json:"content"`
}

func (event Event) GetContent() string {
	var realContent TextContent
	err := json.Unmarshal([]byte(event.Content), &realContent)
	if printErr(err) {
		return realContent.Content
	}
	return ""
}

type FetchSessionMsgsData struct {
	Data struct {
		Messages []Event `json:"messages"`
	} `json:"data"`
}

func (session Session) Request(method string, url string, data SendMsgData) []byte {
	dataByte, err := json.Marshal(data)
	if printErr(err) {
		bodyReader := bytes.NewReader(dataByte)
		client := &http.Client{}
		fmt.Printf("url: %v\n", url)
		req, err := http.NewRequest(method, url, bodyReader)
		if printErr(err) {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
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
	}
	return []byte{}
}

func (session *Session) FetchSessionMsgs(talkerID, sessionType, beginSeqno int64) {
	Url := "https://api.vc.bilibili.com/svr_sync/v1/svr_sync/fetch_session_msgs"
	Params := fmt.Sprintf("?talker_id=%v&session_type=%v&begin_seqno=%v", talkerID, sessionType, beginSeqno)

	var fsmd FetchSessionMsgsData
	json.Unmarshal(session.Request("GET", Url+Params, SendMsgData{}), &fsmd)

	for _, v := range fsmd.Data.Messages {
		if v.SenderUID != session.credential.DedeUserID || !session.exceptSelf {
			switch v.GetContent() {
			case "注册", "register", "token", "/注册", "/register", "/token":
				session.reply(v, "token")
			}
		}
	}
}

func (session *Session) NewSessions(beginTs int64) (gsd GetSessionData) {
	Url := "https://api.vc.bilibili.com/session_svr/v1/session_svr/new_sessions"
	Params := fmt.Sprintf("?begin_ts=%v&build=0&mobi_app=web", beginTs)

	json.Unmarshal(session.Request("GET", Url+Params, SendMsgData{}), &gsd)
	return
}

func (session *Session) GetSessions(sessionType int64) (gsd GetSessionData) {
	Url := "https://api.vc.bilibili.com/session_svr/v1/session_svr/get_sessions"
	Params := fmt.Sprintf("?session_type=%v&group_fold=1&unfollow_fold=0&sort_rule=2&build=0&mobi_app=web", sessionType)

	json.Unmarshal(session.Request("GET", Url+Params, SendMsgData{}), &gsd)
	return
}

type SendMsgData struct {
	MsgSenderUID      int64       `json:"msg[sender_uid]" form:"msg[sender_uid]"`
	MsgReceiverID     int64       `json:"msg[receiver_id]" form:"msg[receiver_id]"`
	MsgReceiverType   int64       `json:"msg[receiver_type]" form:"msg[receiver_type]"`
	MsgMsgType        int64       `json:"msg[msg_type]" form:"msg[msg_type]"`
	MsgMsgStatus      int64       `json:"msg[msg_status]" form:"msg[msg_status]"`
	MsgContent        TextContent `json:"msg[content]" form:"msg[content]"`
	MsgTimestamp      int64       `json:"msg[timestamp]" form:"msg[timestamp]"`
	MsgNewFaceVersion int64       `json:"msg[new_face_version]" form:"msg[new_face_version]"`
	MsgDevID          string      `json:"msg[dev_id]" form:"msg[dev_id]"`
	FromFirework      int64       `json:"from_firework" form:"from_firework"`
	Build             int64       `json:"build" form:"build"`
	MobiApp           string      `json:"mobi_app" form:"mobi_app"`
	CsrfToken         string      `json:"csrf_token" form:"csrf_token"`
	Csrf              string      `json:"csrf" form:"csrf"`
}

func (session Session) SendMsg(ReceiverID int64, content string) {
	Url := "https://api.vc.bilibili.com/web_im/v1/web_im/send_msg"
	data := SendMsgData{
		session.credential.DedeUserID,
		ReceiverID,
		1,
		1,
		0,
		TextContent{content},
		time.Now().Unix(),
		0,
		"94669EA6-CB4B-47B0-874A-42E07DC2145B",
		0,
		0,
		"web",
		session.credential.bili_jct,
		session.credential.bili_jct,
	}
	resp := session.Request("POST", Url, data)
	fmt.Printf("rdata: %v\n", string(resp))
}

func (session Session) reply(event Event, content string) {
	if session.credential.DedeUserID != event.SenderUID {
		session.SendMsg(event.SenderUID, content)
	}
}

func (session *Session) run(except_self bool) {
	// 初始化 只接收开始运行后的新消息
	GSD := session.GetSessions(4)
	for _, v := range GSD.Data.SessionList {
		session.maxSeqno[v.TalkerID] = v.MaxSeqno
	}

	ticker := time.NewTicker(6 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		data := session.NewSessions(session.maxTs)
		for _, v := range data.Data.SessionList {
			if v.SessionTs > session.maxTs {
				session.maxTs = v.SessionTs
			}
			go session.FetchSessionMsgs(v.TalkerID, v.SessionType, session.maxSeqno[v.TalkerID])
			session.maxSeqno[v.TalkerID] = v.MaxSeqno
		}
	}
}

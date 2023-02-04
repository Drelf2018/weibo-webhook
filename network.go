package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

//判断文件夹是否存在
func MakeDir(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
}

// 下载图片到本地并更新数据库
func Download(url, line string) {
	resp, err := http.Get(url)
	panicErr(err)
	defer resp.Body.Close()

	dir := "./image/"
	local := dir + strings.Split(path.Base(url), "?")[0]
	MakeDir(dir)

	file, err := os.Create(local)
	panicErr(err)
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	panicErr(err)

	_, err = UpdatePicture(local, line)
	panicErr(err)
}

// 将 Post 信息 Post 给用户 什么双关
func Webhook(post Post) {
	dataByte, err := json.Marshal(post)
	if printErr(err) {
		bodyReader := bytes.NewReader(dataByte)
		for _, url := range GetUrlsByWatch(post.Type + post.Uid) {
			http.Post(url, "application/json;charset=utf-8", bodyReader)
		}
	}
}

type ApiData struct {
	Code  int64     `json:"code"`
	Error string    `json:"error"`
	Data  []Replies `json:"data"`
}

type Replies struct {
	Member struct {
		Mid   string `json:"mid"`
		Uname string `json:"uname"`
	} `json:"member"`
	Content struct {
		Message string `json:"message"`
	} `json:"content"`
}

// 返回最近回复
var GetReplies func() []Replies

func GetRequest(oid int64) func() []Replies {
	BaseURL := "https://aliyun.nana7mi.link/comment.get_comments(%v,type,1:int).replies"
	QueryVar := "?var=type<-comment.CommentResourceType.DYNAMIC"
	req, _ := http.NewRequest("GET", fmt.Sprintf(BaseURL, oid)+QueryVar, nil)

	return func() []Replies {
		client := &http.Client{}
		resp, err := client.Do(req)
		if !printErr(err) {
			return nil
		}

		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if !printErr(err) {
			return nil
		}

		var Api ApiData
		err = json.Unmarshal(body, &Api)
		if !printErr(err) {
			return nil
		}

		if Api.Code != 0 {
			return nil
		}

		return Api.Data
	}
}

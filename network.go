package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

//判断文件夹是否存在
func MakeDir(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
}

// 解析网址为本地
func Url2Local(url string) string {
	return imageFolder + "/" + strings.Split(path.Base(url), "?")[0]
}

// FileExists 判断一个文件是否存在
//
// 参考 https://blog.csdn.net/leo_jk/article/details/118255913
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// 下载图片到本地
func Download(url string) (local string) {
	local = Url2Local(url)
	if url == "" || FileExists(local) {
		return
	}

	resp := panicSecond(http.Get(url))
	defer resp.Body.Close()

	file := panicSecond(os.Create(local))
	defer file.Close()

	panicSecond(io.Copy(file, resp.Body))

	log.Infof("图片 %v 下载完成", url)
	return
}

func DownloadAll(urls []string, url ...string) {
	for _, u := range append(urls, url...) {
		go Download(u)
	}
}

// 替换
func ReplaceData(text string, post *Post) string {
	return strings.NewReplacer(
		"{mid}", post.Mid,
		"{time}", fmt.Sprint(post.Time),
		"{text}", post.Text,
		"{type}", post.Type,
		"{source}", post.Source,
		"{uid}", post.Uid,
		"{name}", post.Name,
		"{face}", post.Face,
		"{pendant}", post.Pendant,
		"{description}", post.Description,
		"{follower}", post.Follower,
		"{following}", post.Following,
		"{picUrls}", strings.Join(post.PicUrls, ","),
	).Replace(text)
}

// 发送请求
func RequestUser(job Job) {
	// 添加参数
	ploady := make(url.Values)
	for k, v := range job.Data {
		ploady.Set(k, v)
	}

	client := &http.Client{}
	req, _ := http.NewRequest(job.Method, job.Url, strings.NewReader(ploady.Encode()))

	//添加请求头
	for k, v := range job.Headers {
		req.Header.Add(k, v)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if printErr(err) {
		body, err := ioutil.ReadAll(resp.Body)
		if printErr(err) {
			log.Infof("成功向用户 %v 发送请求 %v", job.Url, string(body))
		}
	}
	defer resp.Body.Close()
}

// 将 Post 信息 Post 给用户 什么双关
func Webhook(post *Post) {
	pid := post.Type + post.Uid
	for _, user := range GetAllUsers() {
		if user.File == "" {
			continue
		}
		jobs := GetJobsByUser(user.File, pid)
		for _, job := range jobs {
			matched, err := regexp.MatchString(job.Patten, pid)
			if !printErr(err) || !matched {
				continue
			}

			for k, v := range job.Data {
				job.Data[k] = ReplaceData(v, post)
			}

			go RequestUser(job)
		}
	}
}

type ApiData struct {
	Code int64     `json:"code"`
	Data []Replies `json:"data"`
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
func GetReplies() []Replies {
	BaseURL := "https://aliyun.nana7mi.link/comment.get_comments(%v,type,1:int).replies"
	QueryVar := "?var=type<-comment.CommentResourceType.DYNAMIC"

	resp, err := http.Get(fmt.Sprintf(BaseURL, cfg.Oid) + QueryVar)
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

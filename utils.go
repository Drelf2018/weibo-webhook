package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

// 一条博文包含的信息
type Post struct {
	Mid    string  `form:"mid" json:"mid"`
	Time   float64 `form:"time" json:"time"`
	Text   string  `form:"text" json:"text"`
	Type   string  `form:"type" json:"type"`
	Source string  `form:"source" json:"source"`

	Uid      string `form:"uid" json:"uid"`
	Name     string `form:"name" json:"name"`
	Face     string `form:"face" json:"face"`
	Follow   string `form:"follow" json:"follow"`
	Follower string `form:"follower" json:"follower"`
	Desc     string `form:"description" json:"description"`

	PicUrls []string `form:"picUrls" json:"picUrls"`
	Repost  *Post    `form:"repost,omitempty" json:"repost"`
}

// 过滤函数
func Filter(s []Post, fn func(Post) bool) []Post {
	result := make([]Post, 0, len(s))
	for _, v := range s {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

type config struct {
	User     string
	Password string
	DBname   string
	Debug    bool
}

// 获取命令行参数
func (cfg config) Pasre() string {
	flag.StringVar(&cfg.User, "user", "postgres", "用户名")
	flag.StringVar(&cfg.Password, "password", "postgres", "密码")
	flag.StringVar(&cfg.DBname, "dbname", "postgres", "库名")
	flag.BoolVar(&cfg.Debug, "debug", false, "是否开启 debug 模式")
	flag.Parse()
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.User, cfg.Password, cfg.DBname)
}

type User struct {
	Uid      int64  `form:"uid" json:"uid"`
	Password string `form:"password" json:"password"`
	Token    string
	Level    int64
	Watch    []string `form:"watch" json:"watch"`
	Url      string   `form:"url" json:"url"`
}

// 拼接监控
func (user *User) WatchToValue() string {
	return strings.Join(user.Watch, ",")
}

// 生成随机 token
func (user *User) GetNewToken() string {
	user.Token = uuid.NewV4().String()
	return user.Token
}

func panicErr(err error) bool {
	if err != nil {
		panic(err)
	} else {
		return true
	}
}

func printErr(err error) bool {
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return false
	} else {
		return true
	}
}

// 文本相似性判断
//
// 参考 https://blog.csdn.net/weixin_30402085/article/details/96165537
func SimilarText(first, second string) float64 {
	var similarText func(string, string, int, int) int
	similarText = func(str1, str2 string, len1, len2 int) int {
		var sum, max int
		pos1, pos2 := 0, 0

		// Find the longest segment of the same section in two strings
		for i := 0; i < len1; i++ {
			for j := 0; j < len2; j++ {
				for l := 0; (i+l < len1) && (j+l < len2) && (str1[i+l] == str2[j+l]); l++ {
					if l+1 > max {
						max = l + 1
						pos1 = i
						pos2 = j
					}
				}
			}
		}

		if sum = max; sum > 0 {
			if pos1 > 0 && pos2 > 0 {
				sum += similarText(str1, str2, pos1, pos2)
			}
			if (pos1+max < len1) && (pos2+max < len2) {
				s1 := []byte(str1)
				s2 := []byte(str2)
				sum += similarText(string(s1[pos1+max:]), string(s2[pos2+max:]), len1-pos1-max, len2-pos2-max)
			}
		}

		return sum
	}

	l1, l2 := len(first), len(second)
	if l1+l2 == 0 {
		return 0
	}
	sim := similarText(first, second, l1, l2)
	return float64(sim*2) / float64(l1+l2)
}

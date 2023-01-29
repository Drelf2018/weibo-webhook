package main

import (
	"flag"
	"fmt"
	"strings"

	uuid "github.com/satori/go.uuid"
)

// 一条博文包含的信息
type Post struct {
	Mid    int64  `form:"mid" json:"mid"`
	Time   int64  `form:"time" json:"time"`
	Text   string `form:"text" json:"text"`
	Source string `form:"source" json:"source"`

	Uid      int64  `form:"uid" json:"uid"`
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
	user     string
	password string
	dbname   string
}

// 获取命令行参数
func (cfg config) Pasre() string {
	flag.StringVar(&cfg.user, "user", "postgres", "用户名")
	flag.StringVar(&cfg.password, "password", "postgres", "密码")
	flag.StringVar(&cfg.dbname, "dbname", "postgres", "库名")
	flag.Parse()
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.user, cfg.password, cfg.dbname)
}

type User struct {
	uid      int64
	password string
	token    string
	level    int64
	watch    []string
	url      string
}

// 拼接监控
func (user *User) WatchToValue() string {
	return strings.Join(user.watch, ",")
}

// 生成随机 token
func (user *User) GetNewToken() string {
	user.token = uuid.NewV4().String()
	return user.token
}

func checkErr(err error) bool {
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

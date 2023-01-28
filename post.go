package main

import "database/sql"

// 一条博文包含的信息（暂时）后续还需添加诸如头像、签名等参数
type Post struct {
	Mid  int64  `form:"mid" json:"mid"`
	Time int64  `form:"time" json:"time"`
	Text string `form:"text" json:"text"`
}

// var InsertPost, GetAllPost = GetOperation()
var InsertPost, GetAllPost = GetOperation()
var PostList = GetAllPost()

// 返回给定时间之后的博文
func GetPostByTime(BeginTime int64) []Post {
	return Filter(PostList, func(p Post) bool {
		return (p.Time > BeginTime)
	})
}

// 在 PostList 中添加博文并插入数据库
func (post Post) Save() (sql.Result, error) {
	PostList = append(PostList, post)
	return InsertPost(post)
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

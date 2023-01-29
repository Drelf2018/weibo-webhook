package main

import (
	"fmt"
)

var PostList []Post

func init() {
	PostList = GetAllPost()
}

// 返回给定时间之后的博文
func GetPostByTime(BeginTime int64) []Post {
	return Filter(PostList, func(p Post) bool {
		return (p.Time > BeginTime)
	})
}

// 在 PostList 中添加博文并插入数据库
func (post *Post) Save(token string) int64 {
	level := GetLevelByToken(token)
	fmt.Printf("level: %v\n", level)
	return InsertPost(post)
}

package main

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取 beginTs 时间之后的所有博文
func weibo(c *gin.Context) {
	BeginTime := time.Now().Unix()
	beginTs, ok := c.GetQuery("beginTs")
	if ok && beginTs != "" {
		var err error
		BeginTime, err = strconv.ParseInt(beginTs, 10, 64)
		if err != nil {
			c.JSON(200, gin.H{
				"code":  1,
				"error": err.Error(),
			})
			return
		}
	}
	c.JSON(200, gin.H{
		"code": 0,
		"data": GetPostByTime(BeginTime),
	})
}

// 提交博文
func update(c *gin.Context) {
	token, ok := c.GetQuery("token")
	if ok && token != "" {
		var post Post
		c.Bind(&post)
		_, err := post.Save(token)
		checkErr(err)
	}
}

// 运行 gin 服务器
func main() {
	r := gin.Default()

	r.GET("weibo", weibo)
	r.POST("update", update)

	// listen and serve on 0.0.0.0:8080
	r.Run()
}

package main

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取 beginTs 时间之后的所有博文
func weibo(c *gin.Context) {
	TimeNow := float64(time.Now().Unix() - 5)
	beginTs := c.Query("beginTs")
	if beginTs != "" {
		var err error
		TimeNow, err = strconv.ParseFloat(beginTs, 64)
		if err != nil {
			c.JSON(400, gin.H{
				"code":  1,
				"error": err.Error(),
			})
			return
		}
	}
	c.JSON(200, gin.H{
		"code": 0,
		"data": GetPostByTime(TimeNow),
	})
}

// 提交博文
func update(c *gin.Context) {
	token, ok := c.GetQuery("token")
	if ok && token != "" {
		var post Post
		err := c.Bind(&post)
		if err == nil {
			code, msg := post.Save(token)
			c.JSON(200, gin.H{
				"code": code,
				"data": msg,
			})
		} else {
			c.JSON(400, gin.H{
				"code":  3,
				"error": err.Error(),
			})
		}
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

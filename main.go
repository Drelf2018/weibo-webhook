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

// 注册
func Register(c *gin.Context) {
	var user User
	err := c.Bind(&user)
	if printErr(err) {
		user.GetNewToken()
		user.Level = 2
		_, err := InsertUser(user)
		if printErr(err) {
			c.JSON(200, gin.H{
				"code": 0,
				"data": user.Token,
			})
		} else {
			c.JSON(200, gin.H{
				"code":  1,
				"error": err.Error(),
			})
		}
	} else {
		c.JSON(400, gin.H{
			"code":  2,
			"error": err.Error(),
		})
	}
}

// 运行 gin 服务器
func main() {
	credential := Credential{
		188888131,
	}
	session := Session{0, make(map[int64]int64), true, credential}
	session.run(true)

	// r := gin.Default()

	// r.GET("weibo", weibo)
	// r.POST("update", update)
	// r.POST("register", Register)

	// // listen and serve on 0.0.0.0:8080
	// r.Run()
}

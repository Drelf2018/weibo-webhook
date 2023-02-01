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

// 全局配置
var cfg Config

func init() {
	// 从命令行读取数据库连接参数
	cfg.Pasre()
}

func main() {
	// 启动b站私信监听
	credential := Credential{
		188888131,
	}
	go Session{0, make(map[int64]int64), true, credential}.run(5)

	// 运行 gin 服务器
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.GET("weibo", weibo)
	r.POST("update", update)

	r.Run() // listen and serve on 0.0.0.0:8080
}

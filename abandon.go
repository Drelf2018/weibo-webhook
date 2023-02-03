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
			log.Infof("用户 token: %v %v", token, msg)
			c.JSON(200, gin.H{
				"code": code,
				"data": msg,
			})
		} else {
			log.Infof("用户 token: %v 提交错误：%v", token, err.Error())
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
	flag := false
	if cfg.Credential.DedeUserID == -1 {
		flag = true
		log.Error("uid 读取失败")
	}
	if cfg.Credential.sessdata == "" {
		flag = true
		log.Error("sessdata 读取失败")
	}
	if cfg.Credential.bili_jct == "" {
		flag = true
		log.Error("bili_jct 读取失败")
	}
	if flag {
		panic("检查参数")
	}
}

func main() {
	// 启动b站私信监听
	go Session{1000 * time.Now().UnixMilli(), make(map[int64]int64), true, cfg.Credential}.run(7)

	// 运行 gin 服务器
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.GET("weibo", weibo)
	r.POST("update", update)

	r.Run(":5664") // listen and serve on 0.0.0.0:5664
}

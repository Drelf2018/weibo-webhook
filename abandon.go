package main

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

func GetUID(c *gin.Context) (string, bool) {
	UID, ok := c.GetQuery("uid")
	if !ok {
		c.JSON(400, gin.H{
			"code":  1,
			"error": "UID 获取失败",
		})
		return "", false
	}
	return UID, true
}

func GetToken(c *gin.Context, token string) (string, bool) {
	Token, ok := c.GetQuery("token")
	if !ok || (token != "" && Token != token) {
		c.JSON(400, gin.H{
			"code":  1,
			"error": "Token 获取失败",
		})
		return "", false
	}
	return Token, true
}

// 获取 beginTs 时间之后的所有博文
func GetPost(c *gin.Context) {
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
func UpdatePost(c *gin.Context) {
	if token, ok := GetToken(c, ""); ok {
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

// 注册
func Register(c *gin.Context) {
	UID, ok := GetUID(c)
	if !ok {
		return
	}

	if _, ok := GetToken(c, RandomToken[UID][0]); !ok {
		return
	}

	// 筛选所有该用户的评论 有一个符合即可
	TokenCorrect := false
	for _, r := range Filter(GetReplies(), func(r Replies) bool { return r.Member.Mid == UID }) {
		log.Infof("%v(%v): %v", r.Member.Uname, r.Member.Mid, r.Content.Message)
		if r.Content.Message == RandomToken[UID][1] {
			TokenCorrect = true
		}
	}

	if !TokenCorrect {
		c.JSON(400, gin.H{
			"code":  2,
			"error": "Token 验证失败",
		})
		return
	}

	NumberUID, err := strconv.ParseInt(UID, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{
			"code":  3,
			"error": err.Error(),
		})
		return
	}

	var user User
	err = GetUserByUID(NumberUID, &user)
	if err != nil {
		c.JSON(400, gin.H{
			"code":  4,
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": user.Token,
	})
}

// 随机生成验证用 Token
func GetRandomToken(c *gin.Context) {
	if UID, ok := GetUID(c); ok {
		// 一个用来确定后续请求为同一个人发送 一个用来验证b站UID
		rand.Seed(time.Now().UnixNano())
		size := 100000
		num := (rand.Intn(9)+1)*size + rand.Intn(size)
		RandomToken[UID] = [2]string{uuid.NewV4().String(), strconv.Itoa(num)}
		c.JSON(200, gin.H{
			"code": 0,
			"data": RandomToken[UID],
		})
	}
}

// 登录
func Login(c *gin.Context) {
	if UID, ok := GetUID(c); ok {
		if Token, ok := GetToken(c, ""); ok {
			uid, err := strconv.ParseInt(UID, 10, 64)
			if !printErr(err) {
				c.JSON(400, gin.H{
					"code":  2,
					"error": err.Error(),
				})
				return
			}

			var user User
			err = GetUserByUID(uid, &user)
			if !printErr(err) {
				c.JSON(400, gin.H{
					"code":  3,
					"error": err.Error(),
				})
				return
			}

			if user.Token != Token {
				c.JSON(200, gin.H{
					"code":  4,
					"error": "Token 不正确",
				})
				return
			}

			c.JSON(200, gin.H{
				"code": 0,
				"data": user,
			})
		}
	}
}

func Cors() gin.HandlerFunc {
	c := cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowHeaders:    []string{"Content-Type", "Access-Token", "Authorization"},
	}

	return cors.New(c)
}

// 全局配置
var cfg Config
var RandomToken = make(map[string][2]string)

func init() {
	// 从命令行读取数据库连接参数
	cfg.Pasre()
	// 定义读取评论的函数
	if cfg.Oid == -1 {
		panic("请填写动态oid")
	}
	GetReplies = GetRequest(cfg.Oid)
}

func main() {
	// 运行 gin 服务器
	gin.SetMode(Any(cfg.Debug, gin.DebugMode, gin.ReleaseMode))

	r := gin.Default()

	// 跨域设置
	r.Use(Cors())

	r.GET("login", Login)
	r.GET("post", GetPost)
	r.GET("register", Register)
	r.GET("token", GetRandomToken)

	r.POST("update", UpdatePost)

	r.Run(":5664") // listen and serve on 0.0.0.0:5664
}

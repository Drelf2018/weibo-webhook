package webhook

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/liu-cn/json-filter/filter"
	uuid "github.com/satori/go.uuid"
)

func GetUID(c *gin.Context) (string, bool) {
	UID, ok := c.GetQuery("uid")
	if !ok {
		c.JSON(400, gin.H{
			"code": 1,
			"data": "UID 获取失败",
		})
		return "", false
	}
	return UID, true
}

func GetToken(c *gin.Context, token string) (string, bool) {
	Token, ok := c.GetQuery("token")
	if !ok || (token != "" && Token != token) {
		c.JSON(400, gin.H{
			"code": 1,
			"data": "Token 获取失败",
		})
		return "", false
	}
	return Token, true
}

// 获取 beginTs 时间之后的所有博文
func GetPost(c *gin.Context) {
	TimeNow := float64(time.Now().Unix() - 10)
	EndTime := -1.0
	beginTs := c.Query("beginTs")
	endTs := c.Query("endTs")
	if beginTs != "" {
		var err error
		TimeNow, err = strconv.ParseFloat(beginTs, 64)
		if err != nil {
			c.JSON(400, gin.H{
				"code": 1,
				"data": err.Error(),
			})
			return
		}
	}
	if endTs != "" {
		var err error
		EndTime, err = strconv.ParseFloat(endTs, 64)
		if err != nil {
			c.JSON(400, gin.H{
				"code": 2,
				"data": err.Error(),
			})
			return
		}
	}
	UpdateTime[0] = time.Now().Unix()
	c.JSON(200, gin.H{
		"code":   0,
		"data":   GetPostByTime(int64(TimeNow), int64(EndTime)),
		"poster": UpdateTime,
	})
}

// 提交博文
func UpdatePost(c *gin.Context) {
	if token, ok := GetToken(c, ""); ok {
		user := GetUserByKey("token", token)
		if user.Token != token {
			c.JSON(400, gin.H{
				"code": 2,
				"data": "Token 验证失败",
			})
			return
		}
		var post Post
		err := c.Bind(&post)
		if err == nil {
			post.Empty()
			UpdateTime[user.Uid] = time.Now().Unix()
			log.Infof("用户 %v 提交 %v 级博文: %v", user.Uid, user.Level, post.Text)
			code, msg := post.Save(&user)
			log.Infof("用户 %v %v", user.Uid, msg)
			c.JSON(200, gin.H{
				"code": code,
				"data": msg,
			})
		} else {
			log.Errorf("用户 %v 提交失败: %v", user.Uid, err.Error())
			c.JSON(400, gin.H{
				"code": 3,
				"data": err.Error(),
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
			"code": 2,
			"data": "Token 验证失败",
		})
		return
	}

	NumberUID, err := strconv.ParseInt(UID, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{
			"code": 3,
			"data": err.Error(),
		})
		return
	}

	user := GetUserByKey("uid", NumberUID)
	if user.Uid != 0 {
		c.JSON(200, gin.H{
			"code": 0,
			"data": user.Token,
		})
		return
	}

	newUser, err := NewUserByUID(NumberUID)
	if err != nil {
		c.JSON(400, gin.H{
			"code": 4,
			"data": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": newUser.Token,
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

// 登录前置
func BeforeLogin(c *gin.Context) (*User, int) {
	if UID, ok := GetUID(c); ok {
		if Token, ok := GetToken(c, ""); ok {
			uid, err := strconv.ParseInt(UID, 10, 64)
			if !printErr(err) {
				return nil, 2
			}

			user := GetUserByKey("uid", uid)
			if user.Uid == 0 {
				return nil, 3
			}

			if user.Token != Token {
				return nil, 4
			}

			return &user, 0
		}
	}
	return nil, 1
}

func Login(c *gin.Context) {
	user, err := BeforeLogin(c)
	switch err {
	case 0:
		if user.Uid == 188888131 {
			c.JSON(200, gin.H{
				"code": 0,
				"data": filter.OmitMarshal("login", GetAllUsers()).Interface(),
			})
		} else {
			c.JSON(200, gin.H{
				"code": 0,
				"data": filter.OmitMarshal("login", []User{*user}).Interface(),
			})
		}

	case 2:
		c.JSON(400, gin.H{
			"code": 2,
			"data": "UID 输入错误",
		})
	case 3:
		c.JSON(400, gin.H{
			"code": 3,
			"data": "账号不存在",
		})
	case 4:
		c.JSON(200, gin.H{
			"code": 4,
			"data": "Token 不正确",
		})
	}
}

func Modify(c *gin.Context) {
	user, err := BeforeLogin(c)
	switch err {
	case 0:
		var other User
		err := c.Bind(&other)
		if err != nil {
			c.JSON(200, gin.H{
				"code": 2,
				"data": "获取修改数据失败",
			})
			return
		}
		if other.Uid == user.Uid || user.Uid == 188888131 {
			// 普通用户只允许修改这两项
			other.Update("url", other.Url)
			other.Update("watch", strings.Join(other.Watch, ","))
			if user.Uid == 188888131 {
				other.Update("level", other.Level)
				other.Update("xp", other.XP)
			}
			c.JSON(200, gin.H{
				"code": 0,
				"data": filter.OmitMarshal("login", GetUserByKey("uid", other.Uid)).Interface(),
			})
		} else {
			c.JSON(200, gin.H{
				"code": 3,
				"data": "不能修改别人信息哦",
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
var UpdateTime = make(map[int64]int64)
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

func Run(addr ...string) {
	// 运行 gin 服务器
	gin.SetMode(Any(cfg.Debug, gin.DebugMode, gin.ReleaseMode))

	r := gin.Default()

	// 跨域设置
	r.Use(Cors())

	r.Static("image", "image")
	r.StaticFile("favicon.ico", "image/favicon.ico")

	r.GET("login", Login)
	r.GET("post", GetPost)
	r.GET("register", Register)
	r.GET("token", GetRandomToken)

	r.POST("modify", Modify)
	r.POST("update", UpdatePost)

	r.Run(addr...) // listen and serve on 0.0.0.0:8080
}

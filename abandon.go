package webhook

import (
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liu-cn/json-filter/filter"
	uuid "github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"
)

func ExprCTX(expr bool, c *gin.Context, code int, msg string) bool {
	if expr {
		c.JSON(400, gin.H{
			"code": code,
			"data": msg,
		})
	}
	return expr
}

func ErrorCTX(err error, c *gin.Context, code int) bool {
	if !printErr(err) {
		c.JSON(400, gin.H{
			"code": code,
			"data": err.Error(),
		})
		return true
	}
	return false
}

func GetParams(c *gin.Context) (string, string) {
	Token, ok := c.GetQuery("token")
	if !ok {
		Token = ""
	}

	UID, ok := c.GetQuery("uid")
	if !ok {
		UID = ""
	}

	return UID, Token
}

// 简易登录
func EasyLogin(c *gin.Context) *User {
	_, Token := GetParams(c)
	user := GetUser("token", Token)
	if ExprCTX(user.Token != Token, c, 1, "Token 验证失败") {
		return nil
	}

	// ban
	if user.Level < 0 {
		return nil
	}

	return &user
}

// 严格登录
func StrictLogin(c *gin.Context) *User {
	UID, Token := GetParams(c)
	if ExprCTX(UID == "" || Token == "", c, 1, "参数获取失败") {
		return nil
	}

	uid, err := strconv.ParseInt(UID, 10, 64)
	if err != nil {
		uid = 0
	}

	user := GetUser("uid", uid)

	if ExprCTX(user.Uid == 0, c, 2, "账号不存在") {
		return nil
	}

	if ExprCTX(user.Token != Token, c, 3, "Token 不正确") {
		return nil
	}

	// ban
	if user.Level < 0 {
		return nil
	}

	return &user
}

// 获取 beginTs 时间之后的所有博文
func GetPost(c *gin.Context) {
	// 10 秒的冗余还是太短了啊 没事的 10 秒也很厉害了
	TimeNow := time.Now().Unix() - 30
	var EndTime int64 = -1
	beginTs := c.Query("beginTs")
	endTs := c.Query("endTs")
	if beginTs != "" {
		var err error
		TimeNow, err = strconv.ParseInt(beginTs, 10, 64)
		if ErrorCTX(err, c, 1) {
			return
		}
	}
	if endTs != "" {
		var err error
		EndTime, err = strconv.ParseInt(endTs, 10, 64)
		if ErrorCTX(err, c, 2) {
			return
		}
	}
	UpdateTime[0] = time.Now().Unix()
	c.JSON(200, gin.H{
		"code":   0,
		"data":   SavedPosts.GetPostByTime(TimeNow, EndTime),
		"poster": UpdateTime,
	})
}

// 更新在线状态
func Online(c *gin.Context) {
	user := EasyLogin(c)
	if user == nil {
		return
	}
	timeNow := time.Now().Unix()
	UpdateTime[user.Uid] = timeNow
	c.JSON(200, gin.H{
		"code": 0,
		"data": timeNow,
	})
}

// 提交博文
func UpdatePost(c *gin.Context) {
	user := EasyLogin(c)
	if user == nil {
		return
	}

	var post Post
	err := c.Bind(&post)
	if ErrorCTX(err, c, 3) {
		log.Errorf("用户 %v 提交失败: %v", user.Uid, err.Error())
		return
	}
	post.Empty()
	UpdateTime[user.Uid] = time.Now().Unix()
	log.Infof("用户 %v 提交 %v 级博文: %v", user.Uid, user.Level, post.Text)
	code, msg := post.Save(user)
	log.Infof("用户 %v %v", user.Uid, msg)
	c.JSON(200, gin.H{
		"code": code,
		"data": msg,
	})
}

// 注册
func Register(c *gin.Context) {
	UID, Token := GetParams(c)
	if ExprCTX(UID == "", c, 1, "参数获取失败") {
		return
	}

	if ExprCTX(RandomToken[UID][0] == "", c, 2, "请先获取随机密钥") {
		return
	}

	if ExprCTX(Token != RandomToken[UID][0], c, 3, "你不是该用户！") {
		return
	}

	// 筛选所有该用户的评论 有一个符合即可
	TokenCorrect := false
	for _, r := range Filter(GetReplies(), func(r Replies) bool { return r.Member.Mid == UID }) {
		log.Infof("%v(%v): %v", r.Member.Uname, r.Member.Mid, r.Content.Message)
		if r.Content.Message == RandomToken[UID][1] {
			TokenCorrect = true
			break
		}
	}

	if ExprCTX(!TokenCorrect, c, 4, "Token 验证失败") {
		return
	}

	NumberUID, err := strconv.ParseInt(UID, 10, 64)
	if ErrorCTX(err, c, 5) {
		return
	}

	user := GetUser("uid", NumberUID)
	if user.Uid != 0 {
		c.JSON(200, gin.H{
			"code": 0,
			"data": user.Token,
		})
		return
	}

	newUser, err := NewUser(NumberUID)
	if ErrorCTX(err, c, 4) {
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": newUser.Token,
	})
}

// 随机生成验证用 Token
func GetRandomToken(c *gin.Context) {
	UID, _ := GetParams(c)
	if ExprCTX(UID == "", c, 1, "参数获取失败") {
		return
	}

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

// 登录
func Login(c *gin.Context) {
	user := StrictLogin(c)
	if user == nil {
		return
	}
	users := []User{*user}
	if user.Level <= 1 {
		users = GetAllUsers()
	}
	c.JSON(200, gin.H{
		"code": 0,
		"data": filter.OmitMarshal("login", users).Interface(),
	})
}

// 修改用户信息
func Modify(c *gin.Context) {
	user := StrictLogin(c)
	if user == nil {
		return
	}

	// 普通用户不让改
	if ExprCTX(user.Level > 1, c, 2, "不能修改别人信息哦") {
		return
	}

	var other User
	err := c.Bind(&other)
	if ErrorCTX(err, c, 3) {
		return
	}

	other.Update("xp", other.XP)
	other.Update("level", other.Level)
	c.JSON(200, gin.H{
		"code": 0,
		"data": filter.OmitMarshal("modify", GetUser("uid", other.Uid)).Interface(),
	})
}

// 更新配置文件
func UpdateConfig(c *gin.Context) {
	user := EasyLogin(c)
	if user == nil {
		return
	}

	var yml Yml
	err := c.Bind(&yml)
	if ErrorCTX(err, c, 2) {
		return
	}

	filename := "/" + uuid.NewV4().String() + ".yml"

	file := panicSecond(os.Create(cfg.Resource.Yml + filename))
	defer file.Close()

	enc := yaml.NewEncoder(file)
	panicErr(enc.Encode(yml))

	user.Update("file", filename)

	c.JSON(200, gin.H{
		"code": 0,
		"data": filename,
	})
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 处理请求
		c.Next()
	}
}

// 博文更新时间
var UpdateTime = make(map[int64]int64)

// 随机密钥
var RandomToken = make(map[string][2]string)

func init() {
	// 从命令行读取数据库连接参数
	if cfg.Oid == "" {
		panic("请填写动态oid")
	}

	// 初始化文件夹
	MakeDir(cfg.Resource.Yml)
	MakeDir(cfg.Resource.Image)
}

func Run() {
	// 运行 gin 服务器
	gin.SetMode(Any(cfg.Debug, gin.DebugMode, gin.ReleaseMode))

	r := gin.Default()

	// 跨域设置
	r.Use(Cors())

	r.Static("yml", cfg.Resource.Yml)
	r.Static("image", cfg.Resource.Image)
	r.StaticFile("favicon.ico", cfg.Resource.Image+"/favicon.ico")

	// 解析图片网址并返回文件
	// 参考 https://blog.csdn.net/kilmerfun/article/details/123943070 https://blog.csdn.net/weixin_52690231/article/details/124109518
	r.GET("url/*u", func(c *gin.Context) { c.File(Download(c.Param("u")[1:])) })

	r.GET("login", Login)
	r.GET("post", GetPost)
	r.GET("online", Online)
	r.GET("register", Register)
	r.GET("token", GetRandomToken)

	r.POST("modify", Modify)
	r.POST("update", UpdatePost)
	r.POST("config", UpdateConfig)

	r.Run(cfg.Server.Url + ":" + cfg.Server.Port) // listen and serve on 0.0.0.0:8080
}

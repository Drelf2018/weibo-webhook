package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var log = &logrus.Logger{
	Out: os.Stderr,
	Formatter: &nested.Formatter{
		HideKeys:        true,
		TimestampFormat: time.Kitchen,
	},
	Hooks: make(logrus.LevelHooks),
	Level: logrus.DebugLevel,
}

// 一条博文包含的信息
type Post struct {
	Mid    string  `form:"mid" json:"mid"`
	Time   float64 `form:"time" json:"time"`
	Text   string  `form:"text" json:"text"`
	Type   string  `form:"type" json:"type"`
	Source string  `form:"source" json:"source"`

	Uid      string `form:"uid" json:"uid"`
	Name     string `form:"name" json:"name"`
	Face     string `form:"face" json:"face"`
	Follow   string `form:"follow" json:"follow"`
	Follower string `form:"follower" json:"follower"`
	Desc     string `form:"description" json:"description"`

	PicUrls []string `form:"picUrls" json:"picUrls"`
	Repost  *Post    `form:"repost,omitempty" json:"repost"`
}

// 过滤函数
func Filter[T any](s []T, fn func(T) bool) []T {
	result := make([]T, 0, len(s))
	for _, v := range s {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

// 三目运算符
func Any[T any](Expr bool, TrueReturn, FalseReturn T) T {
	if Expr {
		return TrueReturn
	}
	return FalseReturn
}

// 直接赋值的三目运算符
func AnyTo[T any](Expr bool, Pointer *T, Value T) {
	if Expr {
		*Pointer = Value
	}
}

type Credential struct {
	DedeUserID int64
	sessdata   string
	bili_jct   string
	// buvid3     string
}

func (c Credential) ToCookie() string {
	return fmt.Sprintf("DedeUserID=%v; SESSDATA=%v; bili_jct=%v;", c.DedeUserID, c.sessdata, c.bili_jct)
}

type Config struct {
	User       string
	Password   string
	DBname     string
	DriverName string
	Debug      bool
	Credential Credential
}

// 获取命令行参数
func (cfg *Config) Pasre() {
	flag.StringVar(&cfg.User, "user", "postgres", "用户名")
	flag.StringVar(&cfg.Password, "password", "postgres", "密码")
	flag.StringVar(&cfg.DBname, "dbname", "", "库名")
	flag.BoolVar(&cfg.Debug, "debug", false, "是否开启 debug 模式")
	flag.Int64Var(&cfg.Credential.DedeUserID, "uid", -1, "UID")
	flag.StringVar(&cfg.Credential.sessdata, "sessdata", "", "sessdata")
	flag.StringVar(&cfg.Credential.bili_jct, "bili_jct", "", "bili_jct")
	flag.Parse()
	cfg.DriverName = Any(cfg.DBname == "", "sqlite3", "postgres")
}

// 是否为 SQLite 数据库参数
func (cfg Config) isSQLite() bool {
	return cfg.DriverName == "sqlite3"
}

// Postgres 数据库所需参数
func (cfg Config) Key() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.User, cfg.Password, cfg.DBname)
}

type User struct {
	Uid   int64 `form:"uid" json:"uid"`
	Token string
	Level int64
	XP    int64
	Watch []string `form:"watch" json:"watch"`
	Url   string   `form:"url" json:"url"`
}

func panicErr(err error) bool {
	if err != nil {
		panic(err)
	} else {
		return true
	}
}

func printErr(err error) bool {
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return false
	} else {
		return true
	}
}

// 文本相似性判断
//
// 参考 https://blog.csdn.net/weixin_30402085/article/details/96165537
func SimilarText(first, second string) float64 {
	var similarText func(string, string, int, int) int
	similarText = func(str1, str2 string, len1, len2 int) int {
		var sum, max int
		pos1, pos2 := 0, 0

		// Find the longest segment of the same section in two strings
		for i := 0; i < len1; i++ {
			for j := 0; j < len2; j++ {
				for l := 0; (i+l < len1) && (j+l < len2) && (str1[i+l] == str2[j+l]); l++ {
					if l+1 > max {
						max = l + 1
						pos1 = i
						pos2 = j
					}
				}
			}
		}

		if sum = max; sum > 0 {
			if pos1 > 0 && pos2 > 0 {
				sum += similarText(str1, str2, pos1, pos2)
			}
			if (pos1+max < len1) && (pos2+max < len2) {
				s1 := []byte(str1)
				s2 := []byte(str2)
				sum += similarText(string(s1[pos1+max:]), string(s2[pos2+max:]), len1-pos1-max, len2-pos2-max)
			}
		}

		return sum
	}

	l1, l2 := len(first), len(second)
	if l1+l2 == 0 {
		return 0
	}
	sim := similarText(first, second, l1, l2)
	return float64(sim*2) / float64(l1+l2)
}

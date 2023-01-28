package main

import (
	"database/sql"
	"flag"
	"fmt"
	"strings"

	uuid "github.com/satori/go.uuid"
)

// 一条博文包含的信息（暂时）后续还需添加诸如头像、签名等参数
type Post struct {
	Mid  int64  `form:"mid" json:"mid"`
	Time int64  `form:"time" json:"time"`
	Text string `form:"text" json:"text"`
}

// 过滤函数
func Filter(s []Post, fn func(Post) bool) []Post {
	result := make([]Post, 0, len(s))
	for _, v := range s {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

type config struct {
	user     string
	password string
	dbname   string
}

func (cfg config) Pasre() string {
	flag.StringVar(&cfg.user, "user", "postgres", "用户名")
	flag.StringVar(&cfg.password, "password", "postgres", "密码")
	flag.StringVar(&cfg.dbname, "dbname", "postgres", "库名")
	flag.Parse()
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.user, cfg.password, cfg.dbname)
}

type User struct {
	uid      int64
	password string
	token    string
	level    int64
	watch    []string
	url      string
}

func (user *User) WatchToValue() string {
	return strings.Join(user.watch, ",")
}

func (user *User) ValueToWatch(value string) []string {
	user.watch = strings.Split(value, ",")
	return user.watch
}

func (user *User) GetNewToken() string {
	user.token = uuid.NewV4().String()
	return user.token
}

func checkErr(err error) bool {
	if err != nil {
		panic(err)
	} else {
		return true
	}
}

func Query(db *sql.DB, sql string, fn func(*sql.Rows) bool, args ...any) {
	rows, err := db.Query(sql, args...)
	checkErr(err)
	defer rows.Close()

	// 逐条获取值
	if rows != nil {
		for rows.Next() {
			if fn(rows) {
				break
			}
		}
	}
}

func init() {
	// watch := []string{"1", "2"}
	// user := User{2, "xyw924...", "2", 3, watch, "localhost2"}
	// fmt.Printf("user.GetNewToken(): %v\n", user.GetNewToken())
	// fmt.Printf("user: %v\n", user)
	// InsertUser(user)
	GetUrlByWatch("1")
}

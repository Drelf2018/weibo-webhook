package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var db = openDB()
var insertPost, insertUser *sql.Stmt

// 打开数据库并验证
//
// 参考 http://events.jianshu.io/p/86753f1e5585
func openDB() *sql.DB {
	// 从命令行读取数据库连接参数
	var cfg config
	// fmt.Printf("cfg.Pasre(): %v\n", cfg.Pasre())
	db, err := sql.Open("postgres", cfg.Pasre())

	// 使用sql.Open()创建一个空连接池
	// 注释的这行代码是正式使用 postgresql 数据库
	// 测试的时候用文件 test.db 的 sqlite3 数据库
	// db, err := sql.Open("sqlite3", "./test.db")
	checkErr(err)

	//创建一个具有5秒超时期限的上下文。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//使用PingContext()建立到数据库的新连接，并传入上下文信息，连接超时就返回
	err = db.PingContext(ctx)
	checkErr(err)

	// 返回sql.DB连接池
	return db
}

func init() {
	var err error
	// 初始化表 不存在则新建
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS posts(mid bigint,time bigint,text text)")
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users(uid bigint,password text,token text,level bigint,watch text,url text)")
	checkErr(err)

	// 初始化插入语句
	insertPost, err = db.Prepare("INSERT INTO posts(mid,time,text) VALUES($1,$2,$3)")
	checkErr(err)
	insertUser, err = db.Prepare("INSERT INTO users(uid,password,token,level,watch,url) VALUES($1,$2,$3,$4,$5,$6)")
	checkErr(err)
}

// 向数据库插入一条博文。
func InsertPost(post Post) (sql.Result, error) {
	return insertPost.Exec(post.Mid, post.Time, post.Text)
}

// 向数据库插入一位用户。
func InsertUser(user User) (sql.Result, error) {
	return insertUser.Exec(user.uid, user.password, user.token, user.level, user.WatchToValue(), user.url)
}

// 返回数据库中所有博文。
func GetAllPost() (PostList []Post) {
	Query(db, "select * from posts", func(rows *sql.Rows) bool {
		var post Post
		err := rows.Scan(&post.Mid, &post.Time, &post.Text)
		if err != nil {
			fmt.Println(err)
		} else {
			PostList = append(PostList, post)
		}
		return false
	})
	return
}

// 根据 token 返回级别。
func GetLevelByToken(token string) (level int64) {
	Query(db, "select level from users where token=$1", func(rows *sql.Rows) bool {
		err := rows.Scan(&level)
		if err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}, token)
	return
}

// 根据 watch 返回 url。
func GetUrlByWatch(watch string) {
	Query(db, "select url from users where position($1 in watch) > 0", func(rows *sql.Rows) bool {
		var url string
		err := rows.Scan(&url)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("user: %v\n", url)
		}
		return false
	}, watch)
}

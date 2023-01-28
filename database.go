package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	user     string
	password string
	dbname   string
}

/*
打开数据库并验证

参考 http://events.jianshu.io/p/86753f1e5585
*/
func openDB() (*sql.DB, error) {
	// 从命令行读取数据库连接参数
	var cfg config
	flag.StringVar(&cfg.user, "user", "postgres", "用户名")
	flag.StringVar(&cfg.password, "password", "postgres", "密码")
	flag.StringVar(&cfg.dbname, "dbname", "postgres", "库名")
	flag.Parse()

	// 使用sql.Open()创建一个空连接池
	key := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.user, cfg.password, cfg.dbname)
	fmt.Println(key)

	// 注释的这行代码是正式使用 postgresql 数据库
	// 测试的时候用文件 test.db 的 sqlite3 数据库
	// db, err := sql.Open("postgres", key)
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		return nil, err
	}

	//创建一个具有5秒超时期限的上下文。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//使用PingContext()建立到数据库的新连接，并传入上下文信息，连接超时就返回
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	// 返回sql.DB连接池
	return db, nil
}

/*
获取包装过的数据库操作函数。

SavePost: 向数据库插入一条博文。

GetAllPost: 返回数据库中所有博文。
*/
func GetOperation() (InsertPost func(post Post) (sql.Result, error), GetAllPost func() []Post) {
	// 获取数据库
	db, err := openDB()
	checkErr(err)

	// 初始化表 不存在则新建
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS posts(mid bigint,time bigint,text text)")
	checkErr(err)

	// 初始化插入语句
	insertPost, err := db.Prepare("INSERT INTO posts(mid,time,text) VALUES($1,$2,$3)")
	checkErr(err)

	// 要导出的插入函数
	InsertPost = func(post Post) (sql.Result, error) {
		return insertPost.Exec(post.Mid, post.Time, post.Text)
	}

	// 要导出的获取函数
	GetAllPost = func() (PostList []Post) {
		rows, err := db.Query("select * from posts")
		checkErr(err)
		defer rows.Close()

		// 逐条获取博文
		if rows != nil {
			for rows.Next() {
				var post Post
				if rows.Scan(&post.Mid, &post.Time, &post.Text) != nil {
					fmt.Println(err)
				} else {
					PostList = append(PostList, post)
				}
			}
		}
		return
	}

	return
}

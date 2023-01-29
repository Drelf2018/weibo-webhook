package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var db = openDB()
var insertPic, insertPost, insertUser *sql.Stmt

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
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS pictures(id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,url text,local text)")
	checkErr(err)
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS posts(
		id       bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
		mid      bigint,
		time     bigint,
		text     text,
		source   text,
		picUrls  text,
		repost   text,
		uid      bigint,
		name     text,
		face     text,
		follow   text,
		follower text,
		description text
	)`)
	checkErr(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users(uid bigint,password text,token text,level bigint,watch text,url text)")
	checkErr(err)

	// 初始化插入语句
	insertPic, err = db.Prepare("INSERT INTO pictures(url,local) VALUES($1,$2) returning id")
	checkErr(err)
	insertPost, err = db.Prepare(`
		INSERT INTO posts(mid,time,text,source,picUrls,repost,uid,name,face,follow,follower,description)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning id
	`)
	checkErr(err)
	insertUser, err = db.Prepare("INSERT INTO users(uid,password,token,level,watch,url) VALUES($1,$2,$3,$4,$5,$6)")
	checkErr(err)
}

// 向数据库插入一条博文。
func InsertSinglePost(post *Post, repostID int64) (postID int64) {
	if post == nil {
		return 0
	}

	if insertPost.QueryRow(
		post.Mid,
		post.Time,
		post.Text,
		post.Source,
		SavePictures(post.PicUrls),
		repostID,
		post.Uid,
		post.Name,
		SavePictures([]string{post.Face}),
		post.Follow,
		post.Follower,
		post.Desc,
	).Scan(&postID) != nil {
		return 0
	}
	PostList = append(PostList, *post)
	return
}

// 插入接收到的数据，包含被转发微博
func InsertPost(post *Post) int64 {
	return InsertSinglePost(post, InsertSinglePost(post.Repost, 0))
}

// 向数据库插入一张图片
func InsertPic(url string) (line string) {
	// 判断是否已经保存过
	ForEach(func(rows *sql.Rows) {
		rows.Scan(&line)
	}, "select id from pictures where url=$1", url)
	// 未保存且保存成功后返回行号
	if line == "" && insertPic.QueryRow(url, "Images").Scan(&line) != nil {
		return ""
	} else {
		return
	}
}

// 保存图片合集
func SavePictures(urls []string) string {
	var pids []string
	for _, url := range urls {
		line := InsertPic(url)
		pids = append(pids, line)
		// 异步下载
		go Download(url, line)
	}
	return strings.Join(pids, ",")
}

// 向数据库插入一位用户。
func InsertUser(user User) (sql.Result, error) {
	return insertUser.Exec(user.uid, user.password, user.token, user.level, user.WatchToValue(), user.url)
}

// 返回数据库中所有图片。
func GetAllPictures() (Pictures []string) {
	ForEach(func(rows *sql.Rows) {
		var url string
		err := rows.Scan(&url)
		if printErr(err) {
			Pictures = append(Pictures, url)
		}
	}, "select url from pictures")
	return
}

// 返回数据库中所有博文。
func GetAllPost() (PostList []Post) {
	Pictures := GetAllPictures()
	ForEach(func(rows *sql.Rows) {
		var post Post
		var postID int64
		var PicUrls string
		var repostID int64
		err := rows.Scan(
			&postID,
			&post.Mid,
			&post.Time,
			&post.Text,
			&post.Source,
			&PicUrls,
			&repostID,
			&post.Uid,
			&post.Name,
			&post.Face,
			&post.Follow,
			&post.Follower,
			&post.Desc,
		)
		if printErr(err) {
			// 将配图由序号转为链接
			for _, pid := range strings.Split(PicUrls, ",") {
				PicID, err := strconv.ParseInt(pid, 10, 64)
				if printErr(err) {
					post.PicUrls = append(post.PicUrls, Pictures[PicID-1])
				}
			}
			// 头像 同理
			FaceID, err := strconv.ParseInt(post.Face, 10, 64)
			if printErr(err) {
				post.Face = Pictures[FaceID-1]
			}
			// 添加转发的微博
			if repostID != 0 {
				post.Repost = &PostList[repostID-1]
			}
			PostList = append(PostList, post)
		}
	}, "select * from posts")
	return
}

// 根据 token 返回级别。
func GetLevelByToken(token string) (level int64) {
	ForEach(func(rows *sql.Rows) {
		if printErr(rows.Scan(&level)) {
			fmt.Println(level)
		}
	}, "select level from users where token=$1", token)
	return
}

// 根据 watch 返回 url。
func GetUrlByWatch(watch string) {
	ForEach(func(rows *sql.Rows) {
		var url string
		if printErr(rows.Scan(&url)) {
			fmt.Printf("user: %v\n", url)
		}
	}, "select url from users where position($1 in watch) > 0", watch)
}

func ForEach(fn func(*sql.Rows), cmd string, args ...any) {
	rows, err := db.Query(cmd, args...)
	checkErr(err)
	defer rows.Close()

	// 逐条获取值
	if rows != nil {
		for rows.Next() {
			fn(rows)
		}
	}
}

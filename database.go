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
	uuid "github.com/satori/go.uuid"
)

var db *sql.DB
var insertPic, insertPost, insertUser *sql.Stmt

// 打开数据库并验证
//
// 参考 http://events.jianshu.io/p/86753f1e5585
func init() {
	// 不同数据库的自增语法
	AutoIncrement := Any(cfg.isSQLite(), "INTEGER PRIMARY KEY AUTOINCREMENT", "SERIAL PRIMARY KEY")
	// 不同数据库的位置查找语法
	GetUrlsByWatch = ToGetUrlsByWatch(Any(
		cfg.isSQLite(),
		func(watch string) string {
			return "select url from users where watch Like '%" + watch + "%'"
		},
		func(watch string) string {
			return "select url from users where position(" + watch + " in watch) > 0"
		},
	))
	// 使用sql.Open()创建一个空连接池
	var err error
	db, err = sql.Open(cfg.DriverName, Any(cfg.isSQLite(), "./sqlite3.db", cfg.Key()))
	panicErr(err)

	//创建一个具有5秒超时期限的上下文。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//使用PingContext()建立到数据库的新连接，并传入上下文信息，连接超时就返回
	err = db.PingContext(ctx)
	panicErr(err)

	// 初始化表 不存在则新建
	_, err = db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS pictures(id %v,url text,local text)", AutoIncrement))
	panicErr(err)
	_, err = db.Exec(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS posts(
		id       %v,
		mid      text,
		time     bigint,
		text     text,
		type	 text,
		source   text,
		picUrls  text,
		repost   text,
		uid      text,
		name     text,
		face     text,
		follow   text,
		follower text,
		description text
	)`, AutoIncrement))
	panicErr(err)
	_, err = db.Exec(fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS users(
		uid %v,
		token text,
		level bigint,
		xp bigint,
		watch text,
		url text
	)`, AutoIncrement))
	panicErr(err)

	// 初始化插入语句
	insertPic, err = db.Prepare("INSERT INTO pictures(url,local) VALUES($1,$2) Returning id")
	panicErr(err)
	insertPost, err = db.Prepare(`
		INSERT INTO posts(mid,time,text,type,source,picUrls,repost,uid,name,face,follow,follower,description)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		Returning id
	`)
	panicErr(err)
	insertUser, err = db.Prepare("INSERT INTO users(uid,token,level,xp,watch,url) VALUES($1,$2,$3,$4,$5,$6)")
	panicErr(err)
}

// 插入接收到的数据，包含被转发微博
//
// 我超好巧妙的递归储存
func (post *Post) Insert() (postID int64) {
	if post == nil {
		return 0
	}
	if insertPost.QueryRow(
		post.Mid,
		post.Time,
		post.Text,
		post.Type,
		post.Source,
		SavePictures(post.PicUrls),
		post.Repost.Insert(),
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
		// 异步下载
		go Download(url, line)
		return
	}
}

// 更新图片
func UpdatePic(local, line string) (sql.Result, error) {
	return db.Exec("UPDATE pictures SET local=$1 WHERE id=$2;", local, line)
}

// 保存图片合集
func SavePictures(urls []string) string {
	var pids []string
	for _, url := range urls {
		pids = append(pids, InsertPic(url))
	}
	return strings.Join(pids, ",")
}

// 向数据库插入一位用户。
func (user User) Insert() (sql.Result, error) {
	return insertUser.Exec(user.Uid, user.Token, user.Level, user.XP, strings.Join(user.Watch, ","), user.Url)
}

// 更新用户数据
func (user User) Update(key, value string) (sql.Result, error) {
	return db.Exec(fmt.Sprintf("UPDATE users SET %v='%v' WHERE uid=%v", key, value, user.Uid))
}

// 根据 uid 返回 User 对象
func GetUserByUID(uid int64, user *User) error {
	ForEach(func(rows *sql.Rows) {
		var watch string
		rows.Scan(&user.Uid, &user.Token, &user.Level, &user.XP, &watch, &user.Url)
		user.Watch = strings.Split(watch, ",")
	}, "select * from users where uid=$1", uid)

	if user.Uid != uid {
		user = &User{uid, uuid.NewV4().String(), 5, 0, []string{}, ""}
		_, err := user.Insert()
		if !printErr(err) {
			return fmt.Errorf("储存数据出错")
		}
	}
	return nil
}

// 返回数据库中所有图片。
func GetAllPictures() (Pictures []string) {
	ForEach(func(rows *sql.Rows) {
		var url string
		err := rows.Scan(&url)
		if printErr(err) {
			Pictures = append(Pictures, url)
		}
	}, "select url from pictures order by id")
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
			&post.Type,
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
	}, "select * from posts order by id")
	return
}

// 根据 token 返回级别。
func GetLevelByToken(token string) (level float64) {
	ForEach(func(rows *sql.Rows) {
		if !printErr(rows.Scan(&level)) {
			level = 0
		}
	}, "select level from users where token=$1", token)
	return
}

// 根据 watch 返回 url。
var GetUrlsByWatch func(watch string) (Urls []string)

// 获取 GetUrlsByWatch 函数
func ToGetUrlsByWatch(cmd func(string) string) func(string) []string {
	return func(watch string) (Urls []string) {
		ForEach(func(rows *sql.Rows) {
			var url string
			if printErr(rows.Scan(&url)) {
				Urls = append(Urls, url)
			}
		}, cmd(watch))
		return
	}
}

// 包装后的查询函数
func ForEach(fn func(*sql.Rows), cmd string, args ...any) {
	rows, err := db.Query(cmd, args...)
	panicErr(err)
	defer rows.Close()

	// 逐条获取值
	if rows != nil {
		for rows.Next() {
			fn(rows)
		}
	}
}

package webhook

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	uuid "github.com/satori/go.uuid"
)

var db *sql.DB
var PostStmt, UserStmt *sql.Stmt

type User struct {
	Uid   int64  `form:"uid" json:"uid"`
	Token string `json:"token,omit($any)"`
	Level int64  `form:"level" json:"level"`
	XP    int64  `form:"xp" json:"xp"`
	File  string `form:"file" json:"file"`
}

// 打开数据库并验证
//
// 参考 http://events.jianshu.io/p/86753f1e5585
func init() {
	// 使用sql.Open()创建一个空连接池
	var err error
	db, err = sql.Open(cfg.GetDriver())
	panicErr(err)

	//创建一个具有5秒超时期限的上下文。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//使用PingContext()建立到数据库的新连接，并传入上下文信息，连接超时就返回
	panicErr(db.PingContext(ctx))

	// 初始化表 不存在则新建
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS posts(
		mid      text,
		time     bigint,
		text     text,
		type	 text,
		source   text,

		uid      text,
		name     text,
		face     text,
		pendant  text,
		description text,

		follower  text,
		following text,

		attachment text,
		picUrls    text,
		repost     text
	)`)
	panicErr(err)
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users(
		uid bigint,
		token text,
		level bigint,
		xp bigint,
		file text
	)`)
	panicErr(err)

	PostStmt, err = db.Prepare(`
		INSERT INTO posts(mid,time,text,type,source,uid,name,face,pendant,description,follower,following,attachment,picUrls,repost)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`)
	panicErr(err)

	UserStmt, err = db.Prepare("INSERT INTO users(uid,token,level,xp,file) VALUES($1,$2,$3,$4,$5)")
	panicErr(err)
}

// 插入接收到的数据，包含被转发微博
//
// 我超好巧妙的递归储存
func (post *Post) Insert() string {
	if post == nil || post.Mid == "" {
		return ""
	}
	go Download(post.Face)
	go Download(post.Pendant)
	for _, url := range post.PicUrls {
		go Download(url)
	}
	repostID := post.Repost.Insert()
	mids := ForEach(func(rows *sql.Rows) (mid string) {
		rows.Scan(&mid)
		return
	}, "SELECT mid from posts where mid=$1 and type=$2", post.Mid, post.Type)
	if len(mids) == 0 {
		if _, err := PostStmt.Exec(
			post.Mid,
			post.Time,
			post.Text,
			post.Type,
			post.Source,
			post.Uid,
			post.Name,
			post.Face,
			post.Pendant,
			post.Description,
			post.Follower,
			post.Following,
			strings.Join(post.Attachment, ","),
			strings.Join(post.PicUrls, ","),
			repostID,
		); !printErr(err) {
			return ""
		}

		if post.Type != "weiboComment" {
			SavedPosts.PushSort(*post)
		} else {
			SavedPosts.PushComment(repostID, *post)
		}
	}

	return post.Type + post.Mid
}

// 向数据库插入一位用户。
func (user User) Insert() (sql.Result, error) {
	return UserStmt.Exec(user.Uid, user.Token, user.Level, user.XP, user.File)
}

// 更新用户数据
func (user User) Update(key, value any) (sql.Result, error) {
	switch value := value.(type) {
	case string:
		return db.Exec(fmt.Sprintf("UPDATE users SET %v='%v' WHERE uid=%v", key, value, user.Uid))
	default:
		return db.Exec(fmt.Sprintf("UPDATE users SET %v=%v WHERE uid=%v", key, value, user.Uid))
	}
}

// 新建用户
func NewUserByUID(uid int64) (user *User, err error) {
	user = &User{uid, uuid.NewV4().String(), 5, 0, ""}
	_, err = user.Insert()
	return
}

// 根据 Key 返回 User 对象
func GetUserByKey(key string, val any) (user User) {
	users := ForEach(func(rows *sql.Rows) (user User) {
		rows.Scan(&user.Uid, &user.Token, &user.Level, &user.XP, &user.File)
		return
	}, "select * from users where "+key+"=$1", val)
	if len(users) > 0 {
		return users[0]
	}
	return
}

// 返回所有 User 对象
func GetAllUsers() []User {
	return ForEach(func(rows *sql.Rows) (user User) {
		rows.Scan(&user.Uid, &user.Token, &user.Level, &user.XP, &user.File)
		return
	}, "select * from users")
}

// 返回数据库中所有博文。
func GetAllPost(pl *PostList) []Post {
	return ForEach(func(rows *sql.Rows) (post Post) {
		var Attachment, PicUrls, repostID string
		err := rows.Scan(
			&post.Mid,
			&post.Time,
			&post.Text,
			&post.Type,
			&post.Source,

			&post.Uid,
			&post.Name,
			&post.Face,
			&post.Pendant,
			&post.Description,

			&post.Follower,
			&post.Following,

			&Attachment,
			&PicUrls,
			&repostID,
		)
		if printErr(err) {
			// 分割图片和附件
			if PicUrls == "" {
				post.PicUrls = []string{}
			} else {
				post.PicUrls = strings.Split(PicUrls, ",")
			}
			if Attachment == "" {
				post.Attachment = []string{}
			} else {
				post.Attachment = strings.Split(Attachment, ",")
			}
			post.Comments = []*Post{}
			if pl == nil {
				return
			}
			if post.Type != "weiboComment" {
				// 添加转发的微博
				post.Repost = pl.GetPostByName(repostID)
				// 插入并排序
				pl.PushSort(post)
			} else {
				pl.PushComment(repostID, post)
			}
		}
		return
	}, "select * from posts order by time")
}

// 包装后的查询函数
func ForEach[T any](fn func(*sql.Rows) T, cmd string, args ...any) (result []T) {
	rows, err := db.Query(cmd, args...)
	panicErr(err)
	defer rows.Close()

	// 逐条获取值
	if rows != nil {
		for rows.Next() {
			result = append(result, fn(rows))
		}
	}
	return
}

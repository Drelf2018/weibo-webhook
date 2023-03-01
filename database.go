package webhook

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var PostStmt, UserStmt *sql.Stmt

// 打开数据库并验证
//
// 参考 http://events.jianshu.io/p/86753f1e5585
func init() {
	// 使用sql.Open()创建一个空连接池
	db = panicSecond(sql.Open(cfg.GetDriver()))

	//创建一个具有5秒超时期限的上下文。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//使用PingContext()建立到数据库的新连接，并传入上下文信息，连接超时就返回
	panicErr(db.PingContext(ctx))

	// 初始化表 不存在则新建
	panicSecond(db.Exec(`
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
	)`))

	panicSecond(db.Exec(`
	CREATE TABLE IF NOT EXISTS users(
		uid bigint,
		token text,
		level bigint,
		xp bigint,
		file text
	)`))

	PostStmt = panicSecond(db.Prepare(`
		INSERT INTO posts(mid,time,text,type,source,uid,name,face,pendant,description,follower,following,attachment,picUrls,repost)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`))

	UserStmt = panicSecond(db.Prepare("INSERT INTO users(uid,token,level,xp,file) VALUES($1,$2,$3,$4,$5)"))
}

// 包装后的查询函数
func ForEach[T any](fn func(*sql.Rows) T, cmd string, args ...any) (result []T) {
	rows := panicSecond(db.Query(cmd, args...))
	defer rows.Close()

	// 逐条获取值
	if rows != nil {
		for rows.Next() {
			result = append(result, fn(rows))
		}
	}
	return
}

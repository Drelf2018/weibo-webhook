package webhook

import (
	"database/sql"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

type User struct {
	Uid   int64  `form:"uid" json:"uid"`
	Token string `json:"token,omit($any)"`
	Level int64  `form:"level" json:"level"`
	XP    int64  `form:"xp" json:"xp"`
	File  string `form:"file" json:"file"`
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

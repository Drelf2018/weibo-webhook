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

func (user *User) ReadRow(row *sql.Rows) {
	printErr(row.Scan(&user.Uid, &user.Token, &user.Level, &user.XP, &user.File))
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
func NewUser(uid int64) (user *User, err error) {
	user = &User{uid, uuid.NewV4().String(), 5, 0, ""}
	_, err = user.Insert()
	return
}

// 根据 Key 返回 User 对象
func GetUser(key string, val any) (user User) {
	NewQuery("select * from users where "+key+"=$1", val).ForEach(&user)
	return
}

// 返回所有 User 对象
func GetAllUsers() (users []User) {
	var user User
	NewQuery("select * from users").ForEach(&user, func() bool {
		users = append(users, user)
		return true
	})
	return
}

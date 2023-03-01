package webhook

import (
	"flag"
	"fmt"
)

type Config struct {
	Oid      int64
	User     string
	Password string
	DBname   string
	Debug    bool
}

// 获取命令行参数
func (cfg *Config) Pasre() int64 {
	flag.Int64Var(&cfg.Oid, "oid", -1, "验证动态oid")
	flag.StringVar(&cfg.User, "user", "postgres", "用户名")
	flag.StringVar(&cfg.Password, "password", "postgres", "密码")
	flag.StringVar(&cfg.DBname, "dbname", "", "库名")
	flag.BoolVar(&cfg.Debug, "debug", false, "是否开启 debug 模式")
	flag.Parse()
	return cfg.Oid
}

// 返回驱动器
func (cfg Config) GetDriver() (string, string) {
	if cfg.DBname == "" {
		return "sqlite3", "./sqlite3.db"
	}
	return "postgres", cfg.Key()
}

// Postgres 数据库所需参数
func (cfg Config) Key() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.User, cfg.Password, cfg.DBname)
}

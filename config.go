package main

import (
	"flag"
	"fmt"
)

type Config struct {
	Oid        int64
	User       string
	Password   string
	DBname     string
	DriverName string
	Debug      bool
}

// 获取命令行参数
func (cfg *Config) Pasre() {
	flag.Int64Var(&cfg.Oid, "oid", -1, "验证动态oid")
	flag.StringVar(&cfg.User, "user", "postgres", "用户名")
	flag.StringVar(&cfg.Password, "password", "postgres", "密码")
	flag.StringVar(&cfg.DBname, "dbname", "", "库名")
	flag.BoolVar(&cfg.Debug, "debug", false, "是否开启 debug 模式")
	flag.Parse()
	cfg.DriverName = Any(cfg.DBname == "", "sqlite3", "postgres")
}

// 是否为 SQLite 数据库参数
func (cfg Config) isSQLite() bool {
	return cfg.DriverName == "sqlite3"
}

// Postgres 数据库所需参数
func (cfg Config) Key() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.User, cfg.Password, cfg.DBname)
}

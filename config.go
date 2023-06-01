package webhook

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Oid    string
	Debug  bool
	Server struct {
		Url  string
		Port string
	}
	Database struct {
		Driver   string
		DBname   string
		User     string
		Password string
	}
	Resource struct {
		Image string
		Yml   string
	}
}

// 返回驱动器
func (cfg Config) GetDriver() (string, string) {
	return cfg.Database.Driver, func() string {
		switch cfg.Database.Driver {
		case "sqlite3":
			return cfg.Database.DBname
		case "postgres":
			return cfg.Key()
		default:
			return ""
		}
	}()
}

// Postgres 数据库所需参数
func (cfg Config) Key() string {
	cdb := cfg.Database
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cdb.User, cdb.Password, cdb.DBname)
}

var cfg = GetConfig()

// 获取命令行参数
func FilePath() (filepath string) {
	flag.StringVar(&filepath, "config", "config.yml", "配置文件路径")
	flag.Parse()
	return
}

// 读取配置文件
func GetConfig() (conf Config) {
	yamlFile := panicSecond(ioutil.ReadFile(FilePath()))
	panicErr(yaml.Unmarshal(yamlFile, &conf))
	return
}

package webhook

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Job struct {
	Patten  string            `form:"patten" yaml:"patten"`
	Method  string            `form:"method" yaml:"method"`
	Url     string            `form:"url" yaml:"url"`
	Headers map[string]string `form:"headers" yaml:"headers"`
	Data    map[string]string `form:"data" yaml:"data"`
}

type Yml struct {
	Listening []string `form:"listening" yaml:"listening"`
	Jobs      []Job    `form:"jobs" yaml:"jobs"`
}

func GetYmlByUser(filepath string) (config Yml) {
	data, err := os.ReadFile(filepath)
	if printErr(err) {
		printErr(yaml.Unmarshal(data, &config))
	}
	return
}

func GetJobs(pid string) (jobs []Job) {
	var user User
	NewQuery("select * from users where file != ''").ForEach(&user, func() bool {
		yml := GetYmlByUser(cfg.Resource.Yml + user.File)
		for _, id := range yml.Listening {
			if id == pid {
				jobs = append(jobs, yml.Jobs...)
			}
		}
		return true
	})
	return
}

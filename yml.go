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

func GetJobsByUser(filepath, pid string) []Job {
	yml := GetYmlByUser(ymlFolder + filepath)
	for _, id := range yml.Listening {
		if id == pid {
			return yml.Jobs
		}
	}
	return []Job{}
}

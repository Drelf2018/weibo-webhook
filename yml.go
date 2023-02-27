package webhook

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Job struct {
	Patten  string            `yaml:"patten"`
	Method  string            `yaml:"method"`
	Url     string            `yaml:"url"`
	Headers map[string]string `yaml:"header"`
	Data    map[string]string `yaml:"data"`
}

type Yml struct {
	Listening []string `yaml:"listening"`
	Jobs      []Job    `yaml:"jobs"`
}

func GetYmlByUser(filepath string) (config Yml) {
	data, err := os.ReadFile(filepath)
	if printErr(err) {
		printErr(yaml.Unmarshal(data, &config))
	}
	return
}

func GetJobsByUser(filepath, pid string) []Job {
	yml := GetYmlByUser(filepath)
	for _, id := range yml.Listening {
		if id == pid {
			return yml.Jobs
		}
	}
	return []Job{}
}

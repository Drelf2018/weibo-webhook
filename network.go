package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

//判断文件夹是否存在
func MakeDir(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
}

func Download(url, line string) {
	resp, err := http.Get(url)
	panicErr(err)
	defer resp.Body.Close()

	dir := "./image/"
	local := dir + strings.Split(path.Base(url), "?")[0]
	MakeDir(dir)

	file, err := os.Create(local)
	panicErr(err)
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	panicErr(err)

	_, err = UpdatePic(local, line)
	panicErr(err)
}

func WebhookByPost(post Post) {
	for _, url := range GetUrlsByWatch(post.Type + post.Uid) {
		fmt.Printf("url: %v\n", url)
	}
}

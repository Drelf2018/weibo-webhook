## 微博 webhook

运行时使用 `run.bat` 脚本即可。

```go
go run main.go post.go database.go utils.go --user=postgres --password=postgres --dbname=postgres
```

目录下 `test.py` 为测试脚本，~~用于上传一条博文。~~ 此处格式有重要修改 [详见源码](https://github.com/Drelf2018/weibo-webhook/blob/main/test.py)

```python
import httpx
httpx.post("http://localhost:8080/update", data={"mid": 2, "time": 3, "text": "测试"})
```

测试环境中使用的是 `sqlite3` 数据库，正式使用时考虑使用 `postgresql` 数据库。

```diff
// database.go line: 39
- db, err := sql.Open("sqlite3", "./test.db") 
+ db, err := sql.Open("postgres", key)
```

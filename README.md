## 微博 webhook

运行时使用 `run.bat` 脚本即可。

方括号内为可选参数，是 `PostgreSQL` 的连接参数。

若不填则使用 `SQLite3` 数据库。

```go
go run abandon.go utils.go database.go network.go post.go session.go [--user=postgres --password=postgres --dbname=postgres]
```

目录下 `test.py` 为测试脚本，~~用于上传一条博文。~~ 

此处代码经常修改，以 [最终源码](https://github.com/Drelf2018/weibo-webhook/blob/main/test.py) 为准。

```python
import httpx
httpx.post("http://localhost:8080/update", data={"mid": 2, "time": 3, "text": "测试"})
```

---

> 为什么主函数所在文件叫 `abandon.go` ?

因为我发现一件事，就是本程序的数据都是从 ```database.go``` 中 `init()` 定义的 `db *sql.DB` 读取嘛，如果主函数文件叫 `main.go` 的话，在编译的时候会把 `database.go` 放在较后位置，导致 `post.go` 的 `init()` 从数据库取值的时候找不到。查了不少资料都说是根据文件名排序的，但是 `database.go` 明明在 `post.go` 前面啊，而且交换他们的文件名，再在主文件为 `main.go` 的情况下编译居然又能用了。

<div align="center">



<span style="color:grey">文件名排序图</span>

</div>

仔细观察我们发现，他妈的 `main.go` 隔在 `database.go` 与 `post.go` 之间，大胆假设编译时是从主函数所在文件开始按文件名排序，到底后再从头找到该文件，也就是 `database.go` 是最后编译进去的（猜测），这也能解释为什么 `post.go` 和 `database.go` 交换文件名编译又可以了。所以只要把主文件改名到 `database.go` 前就行了，那我肯定用 `a` 打头啊。`a.go?` 不好听，然后我就选了个中国人特有记忆的第一个单词哈哈。

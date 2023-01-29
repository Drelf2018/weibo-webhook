import httpx
import json
res = httpx.post("http://localhost:8080/update?token=uuid2", data={
    "mid": 100,
	"time": 2,
	"text": "测试",
	"source": "来自iPhone",

    "uid": 434334701,
	"name": "七海",
	"face": "https://xxx.png",
	"follow": "200",
	"follower": "100w",
	"description": "七海娜娜米",

	"picUrls": [
        "https://yun.nana7mi.link/7mi.webp",
        "https://yun.nana7mi.link/ico.webp"
    ],
	"repost": json.dumps({
        "mid": 90,
        "time": 1,
        "text": "被转发",
        "source": "来自Harmony",

        "uid": 188888131,
        "name": "12318",
        "face": "https://xxx.png",
        "follow": "100",
        "follower": "7000",
        "description": "你好李鑫",

        "picUrls": [
            "https://yun.nana7mi.link/7mi.webp",
            "https://yun.nana7mi.link/ico.webp"
        ],
        "repost": None
    })
})
print(res.text)
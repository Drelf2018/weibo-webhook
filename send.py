import httpx
import json

res = httpx.post("http://localhost:8080/register", data={
    "uid": 188888131,
	"password": "testtest",
	"watch": ["weibo434334701"],
	"url": "http://localhost:5664/recv",
})
token = res.json()["data"]

for text in ["测试测试我爱你的程度", "测试测试我爱你的程", "试测试我爱你的程度", "测试测试我爱你的"]:
    res = httpx.post(f"http://localhost:8080/update?token={token}", data={
        "mid": "100",
        "time": 2,
        "text": text,
        "type": "weibo",
        "source": "来自iPhone",

        "uid": "434334701",
        "name": "七海",
        "face": "https://tvax2.sinaimg.cn/crop.0.0.300.300.180/007Raq4zly8h9n3ednt62j308c08caa6.jpg?KID=imgbed,tva&Expires=1675125503&ssig=lPRmOeFleH",
        "follow": "200",
        "follower": "100w",
        "description": "七海娜娜米",

        "picUrls": [
            "https://yun.nana7mi.link/7mi.webp",
            "https://yun.nana7mi.link/ico.webp"
        ],
        "repost": json.dumps({
            "mid": "90",
            "time": 1,
            "text": "被转发",
            "type": "weibo",
            "source": "来自Harmony",

            "uid": "188888131",
            "name": "12318",
            "face": "https://tvax2.sinaimg.cn/crop.0.0.300.300.180/007Raq4zly8h9n3ednt62j308c08caa6.jpg?KID=imgbed,tva&Expires=1675125503&ssig=lPRmOeFleH",
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
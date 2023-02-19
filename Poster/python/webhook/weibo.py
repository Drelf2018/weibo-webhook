from datetime import datetime, timedelta
from typing import List, Dict

from lxml import etree

from .utils import Post, Request, Comments


class WeiboPost(Post):
    @staticmethod
    def created_at(timeText: str) -> int:
        "解析微博时间字段转为时间戳"
        return int(datetime.strptime(timeText, "%a %b %d %H:%M:%S %z %Y").timestamp())

    @classmethod
    def transform(cls: "WeiboPost", mblog: dict):
        user: dict = mblog.get("user", {})
        if user is None: user = dict()
        return {
            "mid": str(mblog["mid"]),
            "time": cls.created_at(mblog["created_at"]),
            "text": mblog["text"],
            "type": "weibo",
            "source": mblog["source"],

            "uid": str(user.get("id")),
            "name": user.get("screen_name"),
            "face": user.get("avatar_hd"),
            "pendant": "",
            "description": user.get("description"),

            "follower": str(user.get("followers_count")),
            "following": str(user.get("follow_count")),

            "attachment": [],
            "picUrls": [p["large"]["url"] for p in mblog.get("pics", [])],
            "repost": mblog.get("retweeted_status")
        }

class WeiboComment(Post):
    @staticmethod
    def created_at(timeText: str) -> int:
        """
        标准化微博发布时间

        参考 https://github.com/Cloud-wish/Dynamic_Monitor/blob/main/main.py#L575
        """
        created_at = datetime.now()
        
        if u"分钟" in timeText:
            minute = timeText[:timeText.find(u"分钟")]
            minute = timedelta(minutes=int(minute))
            created_at -= minute
        elif u"小时" in timeText:
            hour = timeText[:timeText.find(u"小时")]
            hour = timedelta(hours=int(hour))
            created_at -= hour
        elif u"昨天" in timeText:
            day = timedelta(days=1)
            timeText -= day
        elif timeText.count('-') == 1:
            created_at -= timedelta(days=365)    
        
        return int(created_at.timestamp())

    @classmethod
    def transform(cls: "WeiboComment", com: dict):
        user: dict = com.get("user", {})
        if user is None: user = dict()
        pic: str = com.get("pic", {}).get("large", {}).get("url", None)
        return {
            "mid": str(com["id"]),
            "time": WeiboComment.created_at(com["created_at"]),
            "text": com["text"],
            "type": "weiboComment",
            "source": com["source"],

            "uid": str(user.get("id")),
            "name": user.get("screen_name"),
            "face": user.get("profile_image_url"),
            "pendant": "",
            "description": "",

            "follower": str(user.get("followers_count")),
            "following": str(user.get("friends_count")),

            "attachment": com["attachment"],
            "picUrls": [pic] if pic else [],
            "repost": com["repost"]
        }


class WeiboRequest(Request):
    def __init__(self, cookies: str):
        super().__init__(cookies=cookies)
        self.usersComments: Dict[str, Comments] = dict()

    async def posts(self, uid: int | str):
        data = await self.request("GET", f"https://m.weibo.cn/api/container/getIndex?containerid=107603{uid}")
        if data is None:
            return
        for card in data["data"]["cards"][::-1]:
            if card["card_type"] != 9: continue
            yield WeiboPost.parse(card["mblog"])

    async def comments(self, post: WeiboPost):
        data = await self.request("GET", f"https://m.weibo.cn/api/comments/show?id={post.mid}")
        if data is None:
            return

        # 清空储存的评论
        postName = post.type + post.mid
        if self.usersComments.get(post.uid, None) is None:
            self.usersComments[post.uid] = Comments(postName)
        elif self.usersComments[post.uid].post != postName:
            self.usersComments[post.uid] = Comments(postName)

        for com in data["data"]["data"][::-1]:
            mid = str(com["id"])
            if mid in self.usersComments[post.uid].comments: continue

            com["attachment"] = [postName]
            com["repost"] = self.usersComments[post.uid].comments.get(str(com.get("reply_id", "")), None)
            self.usersComments[post.uid].comments[mid] = com
            yield WeiboComment.parse(com)


def parse_text(text: str):
    "获取纯净博文内容"
    span = etree.HTML(f'<div id="post">{text}</div>')
    # 将表情替换为图片链接
    for _span in span.xpath('//div[@id="post"]/span[@class="url-icon"]'):
        alt = _span.xpath('./img/@alt')[0]
        src = _span.xpath('./img/@src')[0]
        _span.insert(0, etree.HTML(f'<p>{alt}: {src}</p>'))

    # 获取这个 span 的字符串形式 并去除 html 格式字符
    text: List[str] = [p.replace(u'\xa0', '').replace('&#13;', '\n') for p in span.xpath('.//text()')]

    # 记录所有 <a> 标签出现的位置
    apos: List[int] = [0]
    for _a in span.xpath('.//a/text()'):
        try:
            apos.append(text.index(_a, apos[-1]))
        except ValueError:
            ...
    else:
        apos.pop(0)
    return text, apos

def get_content(texts: List[str]) -> str:
    return "".join([t.split(": ")[0] if ": " in t else t for t in texts])
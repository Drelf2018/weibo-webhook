from datetime import datetime, timedelta
from typing import List

from lxml import etree

from .utils import Post, Request, logger


class WeiboPost(Post):
    @staticmethod
    def created_at(timeText: str) -> int:
        "解析微博时间字段转为时间戳"
        return int(datetime.strptime(timeText, "%a %b %d %H:%M:%S %z %Y").timestamp())

    @classmethod
    def transform(cls: "WeiboPost", mblog: dict):
        user: dict = mblog["user"]
        return {
            "mid": str(mblog["mid"]),
            "time": cls.created_at(mblog["created_at"]),
            "text": mblog["text"],
            "type": "weibo",
            "source": mblog["source"],

            "uid": str(user["id"]),
            "name": user["screen_name"],
            "face": user["avatar_hd"],
            "pendant": "",
            "description": user["description"],

            "follower": str(user["followers_count"]),
            "following": str(user["follow_count"]),

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
        
        if u"刚刚" in timeText:
            created_at = datetime.now()
        elif u"分钟" in timeText:
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
        user: dict = com["user"]
        reply: str = com.get("reply_text", None)
        pic: str = com.get("pic", {}).get("large", {}).get("url", None)
        return {
            "mid": str(com["id"]),
            "time": WeiboComment.created_at(com["created_at"]),
            "text": com["text"],
            "type": "weiboComment",
            "source": com["source"],

            "uid": str(user["id"]),
            "name": user["screen_name"],
            "face": user["profile_image_url"],
            "pendant": "",
            "description": "",

            "follower": str(user["followers_count"]),
            "following": str(user["friends_count"]),

            "attachment": [reply] if reply else [],
            "picUrls": [pic] if pic else [],
            "repost": None
        }


class WeiboRequest(Request):
    def __init__(self, cookies: str):
        super().__init__(cookies=cookies)

    async def get(self, uid: int | str):
        try:
            res = await self.session.get(f"https://m.weibo.cn/api/container/getIndex?containerid=107603{uid}")
            for card in res.json()["data"]["cards"][::-1]:
                try:
                    if card["card_type"] != 9: continue
                    yield WeiboPost.parse(card["mblog"])
                except Exception as e:
                    logger.error(e)
        except Exception as e:
            logger.error(e)

    async def comment(self, post: WeiboPost):
        try:
            res = await self.session.get(f"https://m.weibo.cn/api/comments/show?id={post.mid}")
            for com in res.json()["data"]["data"][::-1]:
                try:
                    yield WeiboComment.parse(com)
                except Exception as e:
                    logger.error(e)
        except Exception as e:
            logger.error(e)


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
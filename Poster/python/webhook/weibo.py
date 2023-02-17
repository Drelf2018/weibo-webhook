from datetime import datetime
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
            "mid": mblog["mid"],
            "time": cls.created_at(mblog["created_at"]),
            "text": mblog["text"],
            "type": "weibo",
            "source": mblog["source"],

            "uid": user["id"],
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


class WeiboRequest(Request):
    def __init__(self, cookies: str):
        super().__init__(cookies=cookies)

    async def get(self, uid: int | str):
        try:
            res = await self.session.get(f"https://m.weibo.cn/api/container/getIndex?containerid=107603{uid}")
            try:
                for card in res.json()["data"]["cards"][::-1]:
                    if card["card_type"] != 9: continue
                    yield WeiboPost.parse(card["mblog"])
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
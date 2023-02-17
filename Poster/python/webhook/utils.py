import asyncio
import json
import logging
from dataclasses import dataclass
from datetime import datetime
from typing import List

import httpx

logger = logging.getLogger("Poster")
logger.setLevel(logging.INFO)
if not logger.handlers:
    handler = logging.StreamHandler()
    handler.setFormatter(
        logging.Formatter(
            "[Poster][%(asctime)s][%(levelname)s][%(filename)s:%(lineno)d]: %(message)s", "%H:%M:%S"
        )
    )
    logger.addHandler(handler)


def count(start: int = 0):
    " 计数装饰器"
    def inner(fn):
        times = start
        async def wapper(*arg, **kwargs):
            nonlocal times
            logger.info("第 %d 次轮询", times)
            times += 1
            return await fn(*arg, **kwargs)
        return wapper
    return inner


@dataclass
class User:
    level: int
    xp: int
    uid: int
    url: str
    watch: List[str]


@dataclass
class Post:
    mid: str
    time: int
    text: str
    type: str
    source: str

    uid: str
    name: str
    face: str
    pendant: str
    description: str

    follower: str
    following: str

    attachment: List[str]
    picUrls: List[str]

    repost: "Post"

    @classmethod
    def transform(cls: "Post", post: dict):
        "将输入 parse 的数据字典转为 Post 格式"
        return post

    @classmethod
    def parse(cls: "Post", post: dict) -> "Post":
        "递归解析"
        if post is None or len(post) == 0: return None
        post = cls.transform(post)
        return Post(repost=cls.parse(post.pop("repost")), **post)

    @property
    def date(self) -> str:
        "返回规定格式字符串时间"
        return datetime.fromtimestamp(self.time).strftime("%H:%M:%S")

    @property
    def data(self) -> dict:
        "返回可以以 data 发送至后端的数据格式"
        res = dict(self.__dict__)
        if self.repost is None:
            res["repost"] = "null"
        else:
            res["repost"] = json.dumps(self.repost.data)
        return res


class Request:
    "通用请求类"

    BaseHeaders = {
        "Connection": "keep-alive",
        "Accept-Language": "zh-CN,zh;q=0.9",
        "Accept-Encoding": "gzip, deflate, br",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
        "Upgrade-Insecure-Requests": "1",
        "User-Agent": "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.62 Safari/537.36",
    }

    def __init__(self, headers: dict = BaseHeaders, cookies: str = ""):
        if cookies: headers.update("cookie", cookies)
        self.session = httpx.AsyncClient(headers=headers)

    def __del__(self):
        # Use the loop to call async close, then stop/close loop.
        asyncio.get_event_loop().create_task(self.session.aclose())

    async def get(self, url: str, **kwargs):
        return await self.session.get(url, **kwargs)

    async def post(self, url: str, **kwargs):
        return await self.session.post(url, **kwargs)


@dataclass
class Poster:
    """
    usage:

    async with Poster(uid, token, url) as poster:
        await poster.update(post)

    or

    poster = Poster(uid, token, url).login()
    
    await poster.update(post)
    """
    uid: int
    token: str
    baseurl: str
    session: Request = Request()

    async def __aenter__(self): return self.login()
    
    async def __aexit__(self, type, value, trace): ...

    def __del__(self):
        del self.session

    def login(self) -> "Poster":
        "登录"
        try:
            res = httpx.get(f"{self.baseurl}/login", params={
                "uid": self.uid,
                "token": self.token,
            })
            data = res.json()
            if data["code"] == 0:
                self.__vaild = True
                self.users = [User(**u) for u in data["data"]]
                for user in self.users:
                    if user.uid == self.uid:
                        logger.info(f"用户 {user.uid} LV{user.level}({user.xp}/100) 登录成功")
            else:
                raise Exception(data["data"])
        except Exception as e:
            logger.error(e)
            self.__vaild = False
        return self

    async def update(self, post: Post):
        "增"
        if self.__vaild:
            res = await self.session.post(f"{self.baseurl}/update", params={ "token": self.token }, data=post.data)
            data = res.json()
            logger.debug(data["data"])
            # res = await self.session.post(f"https://httpbin.org/post", params={ "token": self.token }, data=post.data)
            # data = res.json()
            # logger.info(data["form"])
        else:
            logger.error("未登录")

    async def post(self, beginTs: int = 0, endTs: int = -1):
        "查"
        res = await self.session.get(f"{self.baseurl}/post", params={ "beginTs": beginTs, "endTs": endTs })
        data = res.json()
        if data["code"] == 0:
            return data["data"]
        else:
            logger.error(data["data"])
            return []


    async def modify(self, user: User) -> User:
        "改"
        try:
            res = await self.session.post(f"{self.baseurl}/modify", params={
                "uid": self.uid,
                "token": self.token,
            }, data=user.__dict__)
            data = res.json()
            if data["code"] == 0:
                return User(**data["data"])
            else:
                raise Exception(data["data"])
        except Exception as e:
            logger.error(e)
            return None
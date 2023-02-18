import asyncio
import datetime
import logging
from dataclasses import dataclass
from typing import List

import httpx
from apscheduler.schedulers.asyncio import AsyncIOScheduler

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


def countLog(name: str = "", start: int = 0):
    "计数装饰器"
    def inner(fn):
        if start < 0: return fn
        times = start
        async def wapper(*arg, **kwargs):
            nonlocal times
            logger.info("%s 第 %d 次轮询", name, times)
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
        return datetime.datetime.fromtimestamp(self.time).strftime("%H:%M:%S")

    @property
    def data(self) -> dict:
        "返回可以以 data 发送至后端的数据格式"
        res = dict(self.__dict__)
        if self.repost is not None:
            res["repost"] = self.repost.data
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
        if cookies: headers.update({"cookie": cookies})
        self.session = httpx.AsyncClient(headers=headers)


class Poster(Request):
    """
    usage:

    async with Poster(uid, token, url) as poster:
        await poster.update(post)

    or

    poster = Poster(uid, token, url)

    await poster.login()

    await poster.update(post)
    """
    
    scheduler = AsyncIOScheduler(timezone="Asia/Shanghai")
    run_time = lambda seconds: datetime.datetime.now() + datetime.timedelta(seconds=seconds)

    def __init__(self, uid: int, token: str, baseurl: str):
        self.uid = uid
        self.token = token
        self.baseurl = baseurl
        
        super().__init__()

    async def __aenter__(self): return await self.login()
    
    async def __aexit__(self, type, value, trace): ...

    async def login(self) -> "Poster":
        "登录"
        try:
            res = await self.session.get(f"{self.baseurl}/login", params={
                "uid": self.uid,
                "token": self.token,
            })
            data = res.json()
            if data["code"] == 0:
                self.__vaild = True
                self.users = [User(**u) for u in data["data"]]
                for user in self.users:
                    if user.uid == self.uid:
                        self.me = user
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
            res = await self.session.post(f"{self.baseurl}/update", params={ "token": self.token }, json=post.data)
            data = res.json()
            if data["code"] != 2:
                logger.info(data["data"])
        else:
            logger.error("未登录")

    async def post(self, beginTs: int = 0, endTs: int = -1):
        "查"
        res = await self.session.get(f"{self.baseurl}/post", params={ "beginTs": int(beginTs), "endTs": int(endTs) })
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
    
    @classmethod
    def add_job(cls: "Poster", fn, start: int = 0, interval: int = 5, name: str = "", count: int = -1, args: list = list(), kwargs: dict = dict()):
        @cls.scheduler.scheduled_job("interval", next_run_time=cls.run_time(start), seconds=interval, args=args, kwargs=kwargs)
        @countLog(name, count)
        async def wapper(*arg, **kwargs):
            return await fn(*arg, **kwargs)

    @classmethod
    def job(cls: "Poster", start: int = 0, interval: int = 5, name: str = "", count: int = -1, args: list = list(), kwargs: dict = dict()):
        "轮询装饰器"
        def inner(fn):
            @cls.scheduler.scheduled_job("interval", next_run_time=cls.run_time(start), seconds=interval, args=args, kwargs=kwargs)
            @countLog(name, count)
            async def wapper(*arg, **kwargs):
                return await fn(*arg, **kwargs)
            return wapper
        return inner

    @classmethod
    def run(cls: "Poster", posters: List["Poster"] = list()):
        async def inner():
            cls.scheduler.start()
            while True:
                await asyncio.sleep(60)
        loop = asyncio.new_event_loop()
        for poster in posters:
            loop.create_task(poster.login())
        loop.run_until_complete(inner())
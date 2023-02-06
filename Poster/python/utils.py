import json
import logging
from dataclasses import dataclass
from datetime import datetime
from typing import List, Optional, Tuple

import httpx

logger = logging.getLogger("Poster")
logger.setLevel(logging.INFO)
if not logger.handlers:
    handler = logging.StreamHandler()
    handler.setFormatter(
        logging.Formatter(
            "[Poster][%(asctime)s][%(levelname)s]: %(message)s", "%H:%M:%S"
        )
    )
    logger.addHandler(handler)


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
    follow: str
    follower: str
    description: str

    picUrls: List[str]

    @staticmethod
    def created_at(timeText: str) -> int:
        return int(datetime.strptime(timeText, "%a %b %d %H:%M:%S %z %Y").timestamp())

    @classmethod
    def parse(cls: "Post", card: dict) -> Tuple[Optional["Post"], Optional[Exception]]:
        try:
            mblog: dict = card["mblog"]
            user: dict = mblog["user"]
            post = Post(
                type = "weibo",
                mid = mblog["mid"],
                text = mblog["text"],
                source = mblog["source"],
                time = cls.created_at(mblog["created_at"]),

                uid = user["id"],
                face = user["avatar_hd"],
                name = user["screen_name"],
                follow = user["follow_count"],
                description = user["description"],
                follower = user["followers_count"],

                picUrls = [p["large"]["url"] for p in mblog.get("pics", [])]
            )
        except Exception as e:
            logger.error(e)
            return None, e
        return post, None

    @property
    def data(self) -> dict:
        res = dict(self.__dict__)
        res["repost"] = res.pop("_Post__repost", "null")
        return res

    def set_repost(self, post: "Post") -> "Post":
        self.__repost = json.dumps(post.data)
        return self


@dataclass
class Poster:
    """
    usage:

    with Poster(uid, token, url) as poster:
        poster.update(post)

    or

    poster = Poster(uid, token, url).login()
    
    poster.update(post)
    """
    uid: int
    token: str
    baseurl: str

    def __enter__(self): return self.login()
    
    def __exit__(self, type, value, trace): ...

    def login(self) -> "Poster":
        try:
            res = httpx.get(f"{self.baseurl}/login", params={
                "uid": self.uid,
                "token": self.token,
            })
            data = res.json()
            if data["code"] == 0:
                self.__vaild = True
                logger.info("用户 {uid} LV{level}({xp}/100) 登录成功".format_map(data["data"]))
            else:
                raise Exception(data["data"])
        except Exception as e:
            logger.error(e)
            self.__vaild = False
        return self

    def update(self, post: Post):
        if self.__vaild:
            res = httpx.post(f"{self.baseurl}/update", params={ "token": self.token }, data=post.data)
            data = res.json()
            logger.info(data["data"])
        else:
            logger.error("未登录")

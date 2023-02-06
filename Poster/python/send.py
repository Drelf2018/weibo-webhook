import httpx

from utils import Post, Poster

with Poster(..., ..., ...) as poster:
    WBUID = 7198559139
    res = httpx.get(f"https://m.weibo.cn/api/container/getIndex?containerid=107603{WBUID}")
    for card in res.json()["data"]["cards"]:
        post, err = Post.parse(card)
        if err is None:
            poster.update(post)
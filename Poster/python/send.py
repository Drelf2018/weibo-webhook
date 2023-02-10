import httpx
from apscheduler.schedulers.blocking import BlockingScheduler
from webhook import Poster, WeiboPost, logger

count = 0
mids = list()
scheduler = BlockingScheduler()
poster = Poster(188888131, "e7158000-4a5d-4cc3-9643-3492bd3ab880", "http://localhost:5664").login()
headers = {
    'Connection': 'keep-alive',
    'Accept-Language': 'zh-CN,zh;q=0.9',
    'Accept-Encoding': 'gzip, deflate, br',
    'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8',
    'Upgrade-Insecure-Requests': '1',
    'User-Agent': 'Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.62 Safari/537.36',
    'cookie': '_T_WM=ceb8107a09ffde92303c2600aafcea3e; SUB=_2A25P9OE_DeRhGeFM7lQU8i7PyjmIHXVtFo93rDV6PUJbktAKLVrskW1NQNxRT3IFuL3bXA569DreXZlw2TSGQXb_; SUBP=0033WrSXqPxfM725Ws9jqgMF55529P9D9WWsM7qnLH5XXeUsRC8WX5b75NHD95QNeo-cSKz7e02fWs4DqcjPi--RiKnXiKnci--4i-zEi-2ReKzpe0nt; SSOLoginState=1659933039'
}

def get(uid: int):
    global count
    logger.info("第 %d 次轮询", count)
    try:
        res = httpx.get(f"https://m.weibo.cn/api/container/getIndex?containerid=107603{uid}", headers=headers)
        for card in res.json()["data"]["cards"][::-1]:
            post = WeiboPost.parse(card["mblog"])
            if post.mid not in mids:
                mids.append(post.mid)
                poster.update(post)
    except Exception as e:
        logger.error(e)
    count += 1

scheduler.add_job(get, "interval", seconds=5, args=[7198559139])
scheduler.start()
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from webhook import Poster, Post, logger, parse_text, get_content, count
import asyncio

scheduler = AsyncIOScheduler()
poster = Poster(188888131, "e7158000-4a5d-4cc3-9643-3492bd3ab880", "http://localhost:5664").login()

@count(1)
async def get():
    try:
        for pdata in await poster.post():
            post = Post.parse(pdata)
            ts, _ = parse_text(post.text)
            content = get_content(ts)
            logger.info(content)
    except Exception as e:
        logger.error(e)

scheduler.add_job(get, "interval", seconds=5)

async def main():
    scheduler.start()
    while True:
        await asyncio.sleep(1)

asyncio.new_event_loop().run_until_complete(main())
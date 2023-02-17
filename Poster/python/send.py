import asyncio
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from webhook import Poster, WeiboRequest, logger, count


scheduler = AsyncIOScheduler()
poster = Poster(188888131, "e7158000-4a5d-4cc3-9643-3492bd3ab880", "http://localhost:5664").login()
session = WeiboRequest()

@scheduler.scheduled_job("interval", seconds=5, args=[7198559139])
@count(1)
async def get(uid: int):
    async for post in session.get(uid):
        await poster.update(post)

async def main():
    scheduler.start()
    while True:
        await asyncio.sleep(1)

asyncio.new_event_loop().run_until_complete(main())
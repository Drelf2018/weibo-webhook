from webhook import Poster, WeiboRequest

poster2 = Poster(188888131, "e7158000-4a5d-4cc3-9643-3492bd3ab880", "http://localhost:5664")
session = WeiboRequest("")

async def weibo(uid: str):
    post = None
    async for post in session.posts(uid):
        await poster.update(post)
    if post is not None:
        async for comment in session.comments(post):
            if comment.uid == uid:
                await poster.update(comment)

Poster.add_job(fn=weibo, name="七海", count=1, start=2, args=[7198559139])
Poster.run(poster)
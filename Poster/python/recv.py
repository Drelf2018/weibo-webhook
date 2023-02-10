from apscheduler.schedulers.blocking import BlockingScheduler
from webhook import Poster, Post, logger, parse_text, get_content

count = 0
scheduler = BlockingScheduler()
poster = Poster(188888131, "e7158000-4a5d-4cc3-9643-3492bd3ab880", "http://localhost:5664").login()

def get():
    global count
    logger.info("第 %d 次轮询", count)
    try:
        for pdata in poster.post():
            post = Post.parse(pdata)
            ts, _ = parse_text(post.text)
            content = get_content(ts)
            print(content)
    except Exception as e:
        logger.error(e)
    count += 1

scheduler.add_job(get, "interval", seconds=5)
scheduler.start()
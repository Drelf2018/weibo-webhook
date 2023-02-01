import uvicorn
from fastapi import FastAPI, Body, Request

app = FastAPI()

@app.post("/recv")
async def fn(req: Request, post=Body()):
    form = await req.body()
    print(str(form))

uvicorn.run(app=app, host="0.0.0.0", port=5664)
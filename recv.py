import uvicorn
from fastapi import FastAPI, Body

app = FastAPI()

@app.post("/recv")
async def fn(post=Body()):
    print(post)

uvicorn.run(app=app, host="0.0.0.0", port=5664)
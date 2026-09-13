import db
from fastapi import FastAPI
import uvicorn
import asyncio

local_queue: list[str] = []

app = FastAPI()

@app.get("/")
def hello():
    print("GET / recieved")
    return {"message": "I'm the indexer!"}

@app.put("/queue")
def enqueue():
    print("PUT /queue RECEIVED")
    return {"message": "PUT request recieved"}

async def index() -> RuntimeError:
    while True:
        print("Still indexing web pages...", flush=True)
        await asyncio.sleep(1.25)

# def main():
#     conn = db.connect()

#     db.disconnect(conn)

async def main() -> None:
    indexing_task = asyncio.create_task(index())

    config = uvicorn.Config(
        app,
        host="0.0.0.0",
        port=3001,
    )
    server = uvicorn.Server(config)

    try:
        await server.serve()
    finally:
        indexing_task.cancel()
        await asyncio.gather(indexing_task, return_exceptions=True)

if __name__ == '__main__':
    asyncio.run(main())
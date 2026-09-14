import db
from fastapi import FastAPI
import uvicorn
import asyncio
import logging
import sys

local_queue: list[str] = []

logging.basicConfig(
    level=logging.INFO,
    stream=sys.stdout,
    format="%(levelname)s %(asctime)s %(name)s: %(message)s",
    force=True,
)

logger = logging.getLogger(__name__)

app = FastAPI()

@app.get("/")
def hello():
    return {"message": "I'm the indexer!"}

@app.post("/queue")
def enqueue():
    return {"message": "PUT request recieved"}

async def index() -> RuntimeError:

    while True:
        logger.info("Still indexing web pages...")
        await asyncio.sleep(1.25)

# def main():

async def main() -> None:
    conn = db.connect()

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
        db.disconnect(conn)

if __name__ == '__main__':
    asyncio.run(main())
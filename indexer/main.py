import psycopg2
from typing import Optional
import db

def main():
    conn = db.connect()

    db.disconnect(conn)

if __name__ == '__main__':
    main()
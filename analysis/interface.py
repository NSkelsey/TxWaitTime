from sqlalchemy import create_engine, Table
from sqlalchemy.schema import MetaData
from redis import Redis

engine = create_engine('postgresql://postgres:obscureref@localhost/txwaittime', echo=False)

dbconn = engine.connect()


class Interface():
    """
    A glue object that handles the connection to postgres
    and redis for us
    """

    def __init__(self):
        self.dbconn = engine.connect()
        self.redis  = Redis(host='localhost', db=7)

        self.redis.incr('runs')

    def __enter__(self):
        return self

    def __exit__(self, t, v, tb):
        self.dbconn.close()
        # I am assume redis knows how to close itself

    def avg_conf_time(self):
        # avg confirmation time for the last week
        avg_conf_time = """
        SELECT * FROM avg_conf_time
        """
        res = self.dbconn.execute(avg_conf_time).fetchall()
        self.redis.set('avg_conf_time', res)

    def conf_rates(self):
        # transaction confirmation rates for the week
        conf_rates = """
        SELECT * FROM conf_rates
        """
        res = dbconn.execute(conf_rates).fetchall()
        self.redis.set('conf_rates', res)

    def pubkey_histogram(self):
        # histogram of p2pkh confirmation times
        pubkey_histogram = """
        SELECT * FROM pubkey_histogram
        """
        res = dbconn.execute(pubkey_histogram).fetchall()
        self.redis.set('pubkey_histogram', res)


if __name__ == '__main__':
    # A hacky test to see if we still work
    with Interface() as glue:
        glue.avg_conf_time()
        glue.conf_rates()
        glue.pubkey_histogram()


import json
from datetime import datetime

from sqlalchemy import create_engine, Table
from sqlalchemy.schema import MetaData

from redis import Redis

engine = create_engine('postgresql://postgres:obscureref@localhost/txwaittime', echo=False)

dbconn = engine.connect()


def unpack(res):
    """ formats a list of returned tuples into a json string """
    lst = []
    for tup in res:
        lst.append(tup[0])
    return json.dumps(lst)


class Interface():
    """
    A glue object that handles the connection to postgres
    and redis for us
    """

    def __init__(self):
        self.dbconn = engine.connect()
        self.redis  = Redis(host='localhost', db=7)

    def __enter__(self):
        return self

    def __exit__(self, t, v, tb):
        self.dbconn.close()
        self.redis.set('latest', datetime.now())
        # I am assuming redis knows how to close itself
        print "Conns closed"

    def avg_conf_time(self):
        # avg confirmation time for the last week
        avg_conf_time = """
        SELECT row_to_json(avg_conf_time) FROM avg_conf_time
        """
        res = self.dbconn.execute(avg_conf_time).fetchall()
        res = unpack(res)
        self.redis.set('avg_conf_time', res)

    def conf_rates(self):
        # transaction confirmation rates for the week
        conf_rates = """
        SELECT row_to_json(conf_rates) FROM conf_rates
        """
        res = dbconn.execute(conf_rates).fetchall()
        res = unpack(res)
        self.redis.set('conf_rates', res)

    def pubkey_histogram(self):
        # histogram of p2pkh confirmation times
        pubkey_histogram = """
        SELECT row_to_json(pubkey_histogram) FROM pubkey_histogram
        """
        res = dbconn.execute(pubkey_histogram).fetchall()
        res = unpack(res)
        self.redis.set('pubkey_histogram', res)


if __name__ == '__main__':
    # A hacky test to see if we still work
    with Interface() as glue:
        glue.avg_conf_time()
        glue.conf_rates()
        glue.pubkey_histogram()


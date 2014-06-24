import datetime

from redis import Redis

from flask import Flask, render_template

app = Flask(__name__)

conn = Redis('localhost', db=7)


@app.route('/')
def home():
    last_update = conn.get('latest')
    avg_conf_times = conn.get('avg_conf_time')
    conf_rates = conn.get('conf_rates')
    pubkey_histogram = conn.get('pubkey_histogram')
    return render_template('home.html', 
            last_update=last_update,
            avg_conf_times=avg_conf_times,
            conf_rates=conf_rates,
            pubkey_histogram=pubkey_histogram,
            )


if __name__ == '__main__':
    app.run(host="0.0.0.0", port=5000, debug=True)

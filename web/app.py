import datetime

from redis import Redis

from flask import Flask, render_template, Response

app = Flask(__name__, static_url_path='/static')

conn = Redis('localhost', db=7)


@app.route('/')
def home():
    last_up = conn.get('latest')
    last_update = datetime.datetime.strptime(last_up, "%Y-%m-%d %H:%M:%S.%f")
    avg_conf_times = conn.get('avg_conf_time')
    conf_rates = conn.get('conf_rates')
    every_histogram = conn.get('every_histogram')
    return render_template('home.html', 
            last_update=last_update,
            avg_conf_times=avg_conf_times,
            conf_rates=conf_rates,
            every_histogram=every_histogram,
            )

@app.route('/json/<string:key>/')
def jsonget(key):
    # pulls key from redis does no processing
    result = conn.get(key)
    return Response(result, mimetype='application/json')


if __name__ == '__main__':
    app.run(host="0.0.0.0", debug=True)

# Run with cmd
from datetime import timedelta

from celery import Celery
from interface import Interface

app = Celery('scheduler',
        broker='redis://localhost:6379',
        backend='redis://localhost:6379/0',
        )

app.conf.CELERYBEAT_SCHEDULE = {
        'compute_summary_stats': {
            'task': 'scheduler.all_statistics',
            'schedule': timedelta(seconds=30),
            },
        }


@app.task
def all_statistics():
    # create glue code obj and destroy it
    with Interface() as elmer:
        elmer.avg_conf_time()
        elmer.conf_rates()
        elmer.pubkey_histogram()


if __name__ == '__main__':
    app.start()

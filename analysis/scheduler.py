# Run with cmd
from datetime import timedelta

from celery import Celery
from celery.utils.log import get_task_logger

from interface import Interface

NAME = 'scheduler'

app = Celery(NAME,
        broker='redis://localhost:6379',
        backend='redis://localhost:6379/0',
        )

app.conf.CELERYBEAT_SCHEDULE = {
        'compute_summary_stats': {
            'task': 'scheduler.all_statistics',
            'schedule': timedelta(seconds=10),
            },
        }


logger = get_task_logger(NAME)


@app.task()
def all_statistics():
    # create glue code obj and destroy it
    with Interface() as elmer:
        logger.info('Connections established')
        logger.info('conf_time starting')
        elmer.avg_conf_time()
        logger.info('conf_rates starting')
        elmer.conf_rates()
        logger.info('pubkey_histogram starting')
        elmer.pubkey_histogram()


if __name__ == '__main__':
    app.start()

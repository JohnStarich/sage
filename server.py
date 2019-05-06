#!/usr/bin/env python3

from datetime import datetime, timedelta
from flask import Flask, Response
from gunicorn.app.base import BaseApplication
from handlers import OfxDownload
from lazy import lazy
from ledger import Ledger
from ofxclient.config import OfxConfig
from os.path import getsize
from pathlib import Path
from rules import RulesFile
from sync import FileLock


app = Flask(__name__)
SYNC = None


class SyncServer(object):
    def __init__(self, ledger: Path, rules: Path, client_config: Path,
                 interval: timedelta = timedelta(minutes = 10)):
        # TODO auto-reload when file updates
        self.ledger_file = ledger
        self.ledger_lock = FileLock(ledger)
        self.sync_interval = interval
        self._rules_file = rules
        self.client_config = OfxConfig(file_name=client_config)
        self._last_sync = None

    @lazy
    def rules(self):
        return RulesFile.from_file(self._rules_file)

    @lazy
    def ledger(self):
        return Ledger.from_file(self.ledger_file)

    @property
    def last_sync(self):
        if self._last_sync is not None:
            return self._last_sync
        if getsize(self.ledger_file) == 0:
            return None
        txn = self.ledger.last_transaction()
        if txn is None:
            return None
        self._last_sync = txn.date
        return self._last_sync

    @last_sync.setter
    def last_sync(self, value):
        self._last_sync = value


class GunicornApp(BaseApplication):
    def __init__(self, app, options=None):
        self.options = options or {}
        self.application = app
        super().__init__()

    def load_config(self):
        config = dict([(key, value) for key, value in self.options.items()
                       if key in self.cfg.settings and value is not None])
        for key, value in config.items():
            self.cfg.set(key.lower(), value)

    def load(self):
        return self.application


def run(sync_server: SyncServer):
    global SYNC
    SYNC = sync_server
    try:
        options = {
            'bind': '%s:%s' % ('0.0.0.0', '8000'),
            # workers must be set to 1 until multiprocess file locking is fixed
            'workers': 1,
        }
        GunicornApp(app, options).run()
    except KeyboardInterrupt:
        pass


@app.route('/ledger')
def get_ledger():
    return str(SYNC.ledger)


def synced_recently() -> (datetime, bool, bool):
    """Returns now, synced-before, and synced-recently"""
    now = datetime.now()
    last_sync = SYNC.last_sync
    if last_sync is None:
        return now, None, False
    return now, last_sync, now - last_sync < SYNC.sync_interval


@app.route('/sync', methods=['POST'])
def sync():
    _, _, recent_sync = synced_recently()
    if recent_sync:
        return ""
    with SYNC.ledger_lock:
        now, sync_time, recent_sync = synced_recently()
        if sync_time is not None:
            if recent_sync:
                return ""
            most_recent_txn = sync_time
        elif len(SYNC.ledger.transactions) > 0:
            most_recent_txn = max(map(lambda t: t.date,
                                      SYNC.ledger.transactions))
        else:
            most_recent_txn = now - timedelta(days=30)
        days_delta = int((now - most_recent_txn).total_seconds() / 3600 / 24)
        days_delta = min(30, days_delta)  # cap at 30 days
        if days_delta < 0:
            raise Exception("Date delta must be positive: %s" % days_delta)
        days_delta = max(1, days_delta)  # ensure a minimum of 1 day is pulled
        handler = OfxDownload(days=days_delta, config=SYNC.client_config)

        def progress_add():
            global SYNC
            with open(SYNC.ledger_file, 'a') as f:
                for statement in handler.transactions():
                    for txn in statement:
                        txn = SYNC.rules.transform(txn)
                        if SYNC.ledger.add(txn):
                            f.write(str(txn))
                            f.write('\n')
                            yield txn
            SYNC.last_sync = now

        return Response(map(str, progress_add()))


if __name__ == '__main__':
    sync_server = SyncServer(
        Path('data/ledger.journal'),
        Path('data/ledger.rules'),
        Path('data/ofxclient.ini'),
        interval=timedelta(seconds=10),
    )
    run(sync_server)

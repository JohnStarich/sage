#!/usr/bin/env python3

from datetime import datetime, timedelta
from flask import Flask, Response
from handlers import OfxDownload
from ledger import Ledger
from ofxclient.config import OfxConfig
from pathlib import Path
from rules import RulesFile
from sync import FileLock
from lazy import lazy


class SyncServer(object):
    def __init__(self, ledger: Path, rules: Path, client_config: Path):
        # TODO auto-reload when file updates
        self.ledger_file = ledger
        self.ledger_lock = FileLock(ledger)
        self.last_sync = None
        self.sync_interval = timedelta(seconds=10)
        self._rules_file = rules
        self.client_config = OfxConfig(file_name=client_config)

    @lazy
    def rules(self):
        return RulesFile.from_file(self._rules_file)

    @lazy
    def ledger(self):
        return Ledger.from_file(self.ledger_file)


app = Flask(__name__)
SYNC = SyncServer(Path('data/ledger.journal'),
                  Path('data/ledger.rules'),
                  Path('data/ofxclient.ini'))


@app.route('/ledger')
def get_ledger():
    return str(SYNC.ledger)


@app.route('/sync', methods=['POST'])
def sync():
    now = datetime.now()
    if SYNC.last_sync is not None and \
            now - SYNC.last_sync < SYNC.sync_interval:
        return ""
    with SYNC.ledger_lock:
        if SYNC.last_sync is not None:
            if now - SYNC.last_sync < SYNC.sync_interval:
                return ""
            most_recent_txn = SYNC.last_sync
        elif len(SYNC.ledger.transactions) > 0:
            most_recent_txn = max(map(lambda t: t.date,
                                      SYNC.ledger.transactions))
        else:
            most_recent_txn = now - timedelta(days=30)
        days_delta = int((now - most_recent_txn).total_seconds() / 3600 / 24)
        days_delta = min(30, days_delta)  # cap at 30 days
        if days_delta < 0:
            raise Exception("Date delta must be positive: %s" % days_delta)
        handler = OfxDownload(days=days_delta, config=SYNC.client_config)

        def progress_add():
            with open(SYNC.ledger_file, 'a') as f:
                for statement in handler.transactions():
                    for txn in statement:
                        txn = SYNC.rules.transform(txn)
                        if txn not in SYNC.ledger:
                            print("Txn %s not in ledger." % txn.postings[0].id)
                        if SYNC.ledger.add(txn):
                            print("Adding txn: %s" % txn.postings[0].id)
                            f.write(str(txn))
                            f.write('\n')
                            yield txn
            SYNC.last_sync = now

        return Response(map(str, progress_add()))

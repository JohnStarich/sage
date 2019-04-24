#!/usr/bin/env python3

from datetime import datetime, timedelta
from flask import Flask, Response
from funcs import filter, func_chain, map
from handlers import OfxDownload
from itertools import chain
from ledger import Ledger
from ofxclient.config import OfxConfig
from pathlib import Path
from rules import RulesFile
from sync import FileLock, apply_rules


app = Flask(__name__)
# TODO auto-reload when file updates
LEDGER_FILE = Path('data/ledger.journal')
LEDGER_LOCK = FileLock(LEDGER_FILE)
with LEDGER_LOCK:
    LEDGER = Ledger.from_file(LEDGER_FILE)
LAST_SYNC = None
SYNC_INTERVAL = timedelta(seconds=10)

RULES = RulesFile.from_file('data/ledger.rules')
CONFIG = OfxConfig(file_name='data/ofxclient.ini')


@app.route('/ledger')
def get_ledger():
    return str(LEDGER)


@app.route('/download')
def download():
    handler = OfxDownload(days=3, config=CONFIG)
    statement_transactions = apply_rules(RULES, handler.transactions())
    txns = func_chain(
        chain.from_iterable(statement_transactions),
        filter(lambda t: t not in LEDGER),
        map(str),
    )
    return Response(txns)


@app.route('/sync', methods=['POST'])
def sync():
    with LEDGER_LOCK:
        now = datetime.now()
        if LAST_SYNC is not None:
            if now - LAST_SYNC < SYNC_INTERVAL:
                return ""
            most_recent_txn = LAST_SYNC
        elif len(LEDGER.transactions) > 0:
            most_recent_txn = min(map(lambda t: t.date, LEDGER.transactions))
        else:
            most_recent_txn = now - timedelta(days=30)
        days_delta = int((now - most_recent_txn).total_seconds() / 3600 / 24)
        days_delta = min(30, days_delta)  # cap at 30 days
        if days_delta < 0:
            raise Exception("Date delta must be positive: %s" % days_delta)
        handler = OfxDownload(days=days_delta, config=CONFIG)

        def progress_add():
            global LAST_SYNC
            with open(LEDGER_FILE, 'a') as f:
                for statement in handler.transactions():
                    for txn in statement:
                        txn = RULES.transform(txn)
                        if LEDGER.add(txn):
                            f.write(str(txn))
                            f.write('\n')
                            yield txn
            LAST_SYNC = now

        return Response(map(str, progress_add()))

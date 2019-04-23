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
ledger_file = Path('data/ledger.journal')
LEDGER = Ledger.from_file(ledger_file)
LEDGER_LOCK = FileLock(ledger_file)
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
    global LAST_SYNC
    # with LEDGER_LOCK:
    if LAST_SYNC is not None:
        if datetime.now() - LAST_SYNC < SYNC_INTERVAL:
            return ""
        most_recent_txn = LAST_SYNC
    else:
        most_recent_txn = LEDGER.transactions[-1].date
    now = datetime.now()
    days_delta = int((now - most_recent_txn).total_seconds() / 3600 / 24)
    days_delta = min(30, days_delta)  # cap at 30 days
    if days_delta < 0:
        raise Exception("Invalid date!!! %s" % days_delta)
    handler = OfxDownload(days=days_delta, config=CONFIG)

    LAST_SYNC = datetime.now()

    def progress_add():
        transactions = func_chain(
            chain.from_iterable(handler.transactions()),
            map(RULES.transform),
        )
        for txn in transactions:
            if LEDGER.add(txn):
                yield txn
    return Response(map(str, progress_add()))

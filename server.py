#!/usr/bin/env python3

from flask import Flask, Response
from funcs import filter, flat_map, func_chain, map
from handlers import OfxDownload
from itertools import chain
from ledger import Ledger
from ofxclient.config import OfxConfig
from pathlib import Path
from rules import RulesFile
from sync import apply_rules


app = Flask(__name__)
# TODO auto-reload when file updates
ledger = Ledger.from_file(Path('data/ledger.journal'))
rules = RulesFile.from_file('data/ledger.rules')
config = OfxConfig(file_name='data/ofxclient.ini')
handler = OfxDownload(days=3, config=config)


@app.route('/ledger')
def get_ledger():
    return str(ledger)


def filter_transactions(txns):
    for txn in txns:
        if txn.id in ledger:
            continue
        in_ledger = False
        for p in txn.postings:
            if p.id in ledger:
                in_ledger = True
                break
        if not in_ledger:
            yield txn


@app.route('/download')
def download():
    statement_transactions = apply_rules(rules, handler.transactions())
    txns = func_chain(
        chain.from_iterable(statement_transactions),
        filter(filter_transactions),
        map(str),
    )
    return Response(txns)

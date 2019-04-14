from itertools import chain
from funcs import func_chain, map, split
from ledger import AccountStatement, LedgerTransaction
from ofxclient import Account as ClientAccount
from ofxclient.config import OfxConfig
from ofxparse import Account, OfxParser, Statement
from typing import Iterable

import sys


class OfxHandler(object):
    def transactions(self) -> Iterable[Iterable[LedgerTransaction]]:
        return func_chain(
            self.accounts(),
            split(self.account_name, self.statement),
            map(AccountStatement.from_pair),
            map(AccountStatement.transactions),
        )

    def account_name(self, account: Account) -> str:
        raise NotImplementedError()

    def statement(self, account: Account) -> Statement:
        raise NotImplementedError()

    def accounts(self) -> Iterable[Account]:
        raise NotImplementedError()


class OfxFiles(OfxHandler):
    def __init__(self, file_names: Iterable[str], config: OfxConfig):
        self._file_names = file_names
        self._account_map = {}
        for account in config.accounts():
            self._account_map[account.number] = account

    def account_name(self, account: Account) -> str:
        account_id = account.account_id
        if account_id in self._account_map:
            return self._account_map[account_id].description
        else:
            return '%s %s' % (
                account.institution.organization,
                account_id,
            )

    def statement(self, account: Account) -> Statement:
        return account.statement

    @staticmethod
    def _parse_ofx(file_name: str):
        with open(file_name, 'r') as f:
            ofx = OfxParser.parse(f)
        if len(ofx.accounts) == 0:
            raise Exception("No accounts found in file: " + file_name)
        return ofx

    def accounts(self) -> Iterable[Account]:
        ofx_files = map(self._parse_ofx, self._file_names)
        accounts = map(lambda ofx: ofx.accounts, ofx_files)
        return chain.from_iterable(accounts)  # flatten accounts


class OfxDownload(OfxHandler):
    """Downloads OFX statements from all accounts in OFX client's config"""

    def __init__(self, days: int, config: OfxConfig):
        self._days = days
        self._accounts = config.accounts()

    def account_name(self, account: Account) -> str:
        return account.description

    def statement(self, account: ClientAccount) -> Statement:
        print("Fetching transactions for %s..." % account.description,
              file=sys.stderr, end=' ')
        statement = account.statement(self._days)
        print("Downloaded transactions.", file=sys.stderr)
        return statement

    def accounts(self) -> Iterable[Account]:
        return self._accounts

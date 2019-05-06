from itertools import chain
from functools import lru_cache
from funcs import func_chain, map, split
from ledger import AccountStatement, LedgerTransaction
from ofxclient import Account as ClientAccount, BankAccount, \
        CreditCardAccount, BrokerageAccount
from ofxclient.config import OfxConfig
from ofxparse import Account, OfxParser, Statement
from typing import Iterable

import sys


@lru_cache()
def _ofx_account_name(account: ClientAccount) -> str:
    institution_str = account.institution.description
    account_str = account.description
    if isinstance(account, BankAccount):
        account_category = 'assets'
        account_str = account_str.lstrip()
        for prefix in [account.institution.description,
                       account.institution.org]:
            if account_str.startswith(prefix):
                account_str = account_str[len(prefix):]
    elif isinstance(account, CreditCardAccount):
        account_category = 'liabilities'
    elif isinstance(account, BrokerageAccount):
        account_category = 'assets:invest'
    else:
        raise Exception("Unknown account type: %s" % type(account))

    account_name = ":".join([
        account_category,
        institution_str.strip(),
        account_str.strip(),
    ])

    if '  ' in account_name:
        account_name = ' '.join(account_name.split())  # Remove extra spaces
    return account_name


class OfxHandler(object):
    def transactions(self) -> Iterable[Iterable[LedgerTransaction]]:
        return func_chain(
            self.accounts(),
            split(lambda a: a, self.account_name, self.statement),
            map(AccountStatement.from_tuple),
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
            institution = account.institution.id
            if institution not in self._account_map:
                self._account_map[institution] = {}
            self._account_map[institution][account.number] = account

    def _lookup_account(self, account: Account) -> ClientAccount:
        institution = account.institution.fid
        account_id = account.account_id
        if institution in self._account_map and \
                account_id in self._account_map[institution]:
            return self._account_map[institution][account_id]
        return None

    def account_name(self, raw: Account) -> str:
        account = self._lookup_account(raw)
        if account is None:
            return '%s %s' % (
                raw.institution.organization,
                raw.account_id,
            )
        return _ofx_account_name(account)

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
        return _ofx_account_name(account)

    def statement(self, account: ClientAccount) -> Statement:
        print("Fetching transactions for %s..." % _ofx_account_name(account),
              file=sys.stderr, end=' ')
        download = account.download(days=self._days)
        parsed = OfxParser.parse(download)
        if 'severity' in parsed.status and \
                parsed.status['severity'] == 'ERROR':
            raise Exception("Error downloading transactions. "
                            "Raw OFX response:\n\n%s" % download.getvalue())
        statement = parsed.account.statement
        print("Downloaded transactions.", file=sys.stderr)
        return statement

    def accounts(self) -> Iterable[Account]:
        return self._accounts

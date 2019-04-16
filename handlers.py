from itertools import chain
from functools import lru_cache
from funcs import func_chain, map, split
from ledger import AccountStatement, LedgerTransaction
from ofxclient import Account as ClientAccount, BankAccount, \
        CreditCardAccount, BrokerageAccount, Institution as ClientInstitution
from ofxclient.config import OfxConfig
from ofxparse import Account, OfxParser, Statement
from typing import Iterable, Union

import sys


@lru_cache(maxsize=20)
def _ofx_account_name(account: Union[Account, ClientAccount]) -> str:
    if isinstance(account, Account):
        institution = ClientInstitution.deserialize(
                account.institution.__dict__)
        account = ClientAccount.from_ofxparse(account, institution)
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
            account = self._account_map[account_id]
            return _ofx_account_name(account)
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
        return _ofx_account_name(account)

    def statement(self, account: ClientAccount) -> Statement:
        print("Fetching transactions for %s..." % _ofx_account_name(account),
              file=sys.stderr, end=' ')
        statement = account.statement(self._days)
        print("Downloaded transactions.", file=sys.stderr)
        return statement

    def accounts(self) -> Iterable[Account]:
        return self._accounts

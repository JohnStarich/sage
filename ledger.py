from decimal import Decimal
from funcs import filter, flat_map, func_chain, map
from itertools import chain, groupby
from ofxparse import Account, Statement, Transaction
from pathlib import Path
from typing import Callable, Collection, Iterable, Tuple


class Ledger(set):
    def __init__(self, lines: Iterable[str]):
        super().__init__(Ledger._parse_lines(lines))

    @staticmethod
    def from_file(path: Path):
        with path.open('r') as f:
            return Ledger(f)

    @staticmethod
    def _parse_lines(lines):
        id_tags = func_chain(
            lines,
            map(lambda line: line.split(';', maxsplit=1)),
            filter(lambda tokens: len(tokens) > 1),
            map(lambda tokens: tokens[1]),
            flat_map(lambda comment: comment.split(',')),
            map(str.strip),
            filter(lambda tag: tag.startswith('id:')),
            map(lambda txn_id: txn_id[len('id:'):]),
            map(str.lstrip),
        )
        for id_tag in id_tags:
            yield str(id_tag)


class LedgerPosting(object):
    def __init__(self, account: str, amount: Decimal, id: str = None,
                 balance: Decimal = None, comment: str = None,
                 currency: str = '$'):
        self.id = id
        self.account = account
        self.amount = amount
        self.balance = balance
        self.comment = comment
        self.currency = currency

    def str_format(self, account_len, amount_len):
        full_fmt = '%-{}s  %s%s%s'.format(account_len)
        amount = ' %{}s'.format(amount_len + 2)
        if self.amount is not None:
            amount = amount % ('$ ' + str(self.amount))
        else:
            amount = amount % ''
        if self.balance is not None:
            balance = ' = $ %s' % self.balance
        else:
            balance = ''
        if self.comment is not None:
            comment = '  ; ' + self.comment
        else:
            comment = ''
        return full_fmt % (self.account, amount, balance, comment)

    @staticmethod
    def format_table(postings: Collection['LedgerPosting']) -> Iterable[str]:
        account_len = max(func_chain(
            postings,
            map(lambda p: p.account),
            map(str),
            map(len),
        ))
        amount_len = max(func_chain(
            postings,
            map(lambda p: p.amount),
            filter(lambda a: a is not None),
            map(str),
            map(len),
        ))

        return map(lambda p: p.str_format(account_len, amount_len), postings)


class LedgerTransaction(object):
    def __init__(self,
                 postings: Collection[LedgerPosting],
                 comment=None,
                 date=None,
                 description=None):
        if len(postings) < 2:
            raise ValueError("Must provide at least two postings.")
        accounts = list(map(lambda p: p.account, postings))
        amounts = list(map(lambda p: p.amount, postings))
        if len(accounts) - len(amounts) not in (0, 1):
            raise ValueError("Number of accounts must be one higher than "
                             "or equal to the number of amounts:"
                             "\n\tAccounts: {}\n\tAmounts: {}"
                             .format(accounts, amounts))
        if len(amounts) == 0:
            raise ValueError("Must provide at least one amount.")

        self.postings = postings
        self.comment = comment
        self.date = date
        self.description = description

    @staticmethod
    def from_ofxparse(account: Account, account_name: str, raw: Transaction,
                      balance: Decimal) -> 'LedgerTransaction':
        # Follows FITID recommendation from OFX 102 Section 3.2.1
        fit_id = ''.join([
            account.institution.fid,
            account.account_id,
            raw.id,
        ])
        # clean ID for hledger tags
        fit_id = fit_id.replace(",", "_").replace(":", "_")
        postings = [
            LedgerPosting(
                id=fit_id,
                account=account_name,
                amount=raw.amount,
                balance=balance,
                comment='id:' + fit_id,
            ),
            LedgerPosting(
                id=None,
                account=None,
                amount=-raw.amount,
            ),
        ]
        return LedgerTransaction(
            postings=postings,
            date=raw.date,
            description=raw.payee,
        )

    def date_str(self) -> str:
        return self.date.strftime('%Y/%m/%d')

    def __str__(self):
        comment = ""
        if self.comment is not None:
            comment = "; " + self.comment
        postings = LedgerPosting.format_table(self.postings)

        header = "{date} {description}{comment}".format(
            date=self.date_str(),
            comment=comment,
            description=self.description,
        )
        return "{header}\n    {postings}\n".format(
            header=header,
            postings='\n    '.join(postings),
        )

    def __lt__(self, other):
        return self.date < other.date


class AccountStatement(object):
    """
    An account-statement pair. Used for parsing transactions and
    applying balance assertions.
    """

    def __init__(self, account: Account, account_name: str,
                 statement: Statement):
        self.account = account
        self.account_name = account_name
        self.statement = statement

    @staticmethod
    def from_tuple(tup: Tuple[Account, str, Statement]) -> 'AccountStatement':
        account, account_name, statement = tup
        return AccountStatement(account, account_name, statement)

    def transactions(self) -> Iterable[LedgerTransaction]:
        groups = groupby(self.statement.transactions,
                         key=lambda t: t.date <= self.statement.balance_date)
        before = []
        after = []
        for key, group in groups:
            if key is True:
                before.extend(group)
            else:
                after.extend(group)

        before.sort(key=lambda t: t.date, reverse=True)
        before = list(map(
            self._parse_balance_transaction(after=False),
            before,
        ))
        before.reverse()

        after.sort(key=lambda t: t.date)
        after = map(
            self._parse_balance_transaction(after=True),
            after,
        )

        return list(chain(before, after))

    def _parse_balance_transaction(
            self, after: bool) -> Callable[[Transaction], LedgerTransaction]:
        """
        Propagates the statement balance going forward or backward in time.
        Call the returned func with transactions in chronological order
        (after=True) or in reverse (after=False) to propagate the balance
        correctly.
        """
        balance = self.statement.balance

        def parse_transaction(raw: Transaction) -> LedgerTransaction:
            nonlocal balance
            t = LedgerTransaction.from_ofxparse(
                    self.account, self.account_name, raw, balance)
            if after is True:
                balance += raw.amount
            else:
                balance -= raw.amount
            return t
        return parse_transaction

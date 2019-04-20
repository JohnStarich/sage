from datetime import datetime
from decimal import Decimal, InvalidOperation
from funcs import filter, flat_map, func_chain, map
from itertools import chain, groupby
from ofxclient import Account as ClientAccount
from ofxparse import Account, Statement, Transaction
from pathlib import Path
from typing import Callable, Collection, Dict, Iterable, Tuple, \
        Union


class Ledger(object):
    def __init__(self, lines: Iterable[str]):
        self.transactions = list(Ledger._parse_lines(lines))
        txn_ids = chain(
            func_chain(
                self.transactions,
                flat_map(lambda t: t.postings),
                map(lambda p: p.id),
                filter(lambda i: i is not None),
            ),
            func_chain(
                self.transactions,
                map(lambda t: t.id),
                filter(lambda i: i is not None),
            ),
        )
        self._transaction_ids = set(txn_ids)

    def __contains__(self, item):
        if item is None:
            return False
        if isinstance(item, str):
            return item in self._transaction_ids
        if isinstance(item, LedgerTransaction):
            return item.id in self._transaction_ids or \
                any(map(lambda p: p.id in self._transaction_ids,
                        item.postings))
        if isinstance(item, LedgerPosting):
            return item.id in self._transaction_ids
        return False

    @staticmethod
    def from_file(path: Path):
        with path.open('r') as f:
            return Ledger(f)

    @staticmethod
    def _parse_lines(lines: Iterable[str]
                     ) -> Iterable['LedgerTransaction']:
        while True:
            try:
                line = next(lines)
                while len(line) == 0 or line.isspace():
                    line = next(lines)
                lines = chain([line], lines)
            except StopIteration:
                return
            transaction, lines = LedgerTransaction.parse_lines(lines)
            if transaction is None:
                return
            yield transaction

    def __str__(self):
        return '\n'.join(map(str, self.transactions))


def format_tags(tags: Dict[str, str]):
    if len(tags) == 0:
        return ''
    if len(tags) > 1:
        print("TAGS! " + str(tags))
    items = sorted(tags.items(), key=lambda i: i[0])
    return " " + ", ".join(
        map(lambda tup: "%s: %s" % tup, items),
    )


class LedgerPosting(object):
    def __init__(self, account: str, amount: Decimal,
                 balance: Decimal = None, currency: str = '$',
                 comment: str = None,
                 tags: Dict[str, str] = dict()):
        self.account = account
        self.amount = amount
        self.balance = balance
        self.comment = comment
        self.currency = currency
        self.tags = tags

    @property
    def id(self):
        return self.tags.get('id', None)

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
        comment = ''
        if self.comment is not None:
            comment = self.comment
        if len(self.tags) > 0:
            comment += format_tags(self.tags)
        if comment != '':
            comment = '  ; ' + comment
        return full_fmt % (self.account, amount, balance, comment)

    @staticmethod
    def parse_line(line: str):
        tokens = line.split('  ', maxsplit=1)
        if len(tokens) != 2:
            raise Exception("Invalid posting: account must be"
                            " separated from amount by 2+ spaces")
        account = tokens[0]

        tokens = tokens[1].split(';', maxsplit=1)
        amount_str = tokens[0].strip()
        comment = None
        if len(tokens) > 1:
            comment = tokens[1].lstrip()

        balance_str = None
        if '=' in amount_str:
            amount_str, balance_str = map(str.strip,
                                          amount_str.split('=', maxsplit=1))
            if amount_str == "":
                raise Exception("Postings can't have a balance assertion "
                                "without an amount: " + balance_str)
        amount = None
        balance = None
        if amount_str != "":
            amount = LedgerPosting._parse_amount(amount_str)
        if balance_str is not None:
            balance = LedgerPosting._parse_amount(balance_str, desc="balance")
        tags = {}
        if comment is not None and ':' in comment:
            # Trim off regular comment
            comment_end = comment.rfind(' ', 0, comment.find(':'))
            full_comment = comment
            comment = full_comment[:comment_end]
            tag_strs = full_comment[comment_end+1:].split(',')
            for tag in tag_strs:
                if ':' not in tag:
                    raise Exception("Invalid tag format: " + tag)
                key, value = tag.split(':', maxsplit=1)
                if ' ' in key:
                    raise Exception("Tag keys must not contain"
                                    " spaces: " + key)
                tags[key] = value

        return LedgerPosting(
            account=account,
            amount=amount,
            balance=balance,
            comment=comment,
            tags=tags,
        )

    @staticmethod
    def _parse_amount(amount, desc="amount"):
        amount = amount.lstrip('$ \t').rstrip().replace(",", "")
        invalid = False
        try:
            amount_dec = Decimal(amount)
        except InvalidOperation:
            invalid = True
        if invalid or amount_dec.is_nan():
            raise Exception("Invalid %s for posting: %s" % (desc, amount))
        return amount_dec

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
                 payee=None,
                 tags: Dict[str, str] = dict()):
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
        self.payee = payee
        self.tags = tags

    @property
    def id(self):
        return self.tags.get('id', None)

    @staticmethod
    def parse_lines(lines: Iterable[str]) -> 'LedgerTransaction':
        try:
            line = next(lines)
        except StopIteration:
            return None, lines
        payee_line = line.rstrip()
        tokens = payee_line.split(maxsplit=1)
        if len(tokens) != 2:
            raise Exception("Invalid transaction payee line: " + line)
        date_str, payee = tokens
        date = datetime.strptime(date_str, "%Y/%m/%d")
        comment = None
        if ';' in payee:
            payee, comment = payee.split(';', maxsplit=1)
        postings = []
        try:
            line = next(lines)
        except StopIteration:
            raise Exception("Incomplete transaction at end of ledger.")
        while line.startswith(' ') or line.startswith('\t'):
            postings.append(LedgerPosting.parse_line(line.strip()))
            try:
                line = next(lines)
            except StopIteration:
                break
        if len(postings) < 2:
            raise Exception(
                "Transactions must have at least two postings:\nPayee: {}"
                "\nPostings:\n    {}".format(payee_line,
                                             '\n    '.join(postings)))
        return LedgerTransaction(
            payee=payee,
            postings=postings,
            comment=comment,
            date=date,
        ), chain([line], lines)

    @staticmethod
    def from_ofxparse(account: Union[Account, ClientAccount],
                      account_name: str, raw: Transaction,
                      balance: Decimal) -> 'LedgerTransaction':
        if isinstance(account, Account):
            account_id = account.account_id
            institution_id = account.institution.fid
        elif isinstance(account, ClientAccount):
            account_id = account.number
            institution_id = account.institution.id
        # Follows FITID recommendation from OFX 102 Section 3.2.1
        fit_id = '-'.join([
            institution_id,
            account_id,
            raw.id,
        ])
        # clean ID for hledger tags
        fit_id = fit_id.replace(",", "_").replace(":", "_")
        postings = [
            LedgerPosting(
                account=account_name,
                amount=raw.amount,
                balance=balance,
                tags={'id': fit_id},
            ),
            LedgerPosting(
                account=None,
                amount=-raw.amount,
            ),
        ]
        return LedgerTransaction(
            postings=postings,
            date=raw.date,
            payee=raw.payee,
        )

    def date_str(self) -> str:
        return self.date.strftime('%Y/%m/%d')

    def __str__(self):
        comment = ""
        if self.comment is not None:
            comment = self.comment
        if len(self.tags) > 0:
            comment += format_tags(self.tags)
        if comment != "":
            comment = "  ; " + comment
        postings = LedgerPosting.format_table(self.postings)

        header = "{date} {payee}{comment}".format(
            date=self.date_str(),
            comment=comment,
            payee=self.payee,
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

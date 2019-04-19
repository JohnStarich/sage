from itertools import chain
from ledger import LedgerTransaction
from typing import Iterator

import re


class RulesFile(object):
    def __init__(self, lines: Iterator[str]):
        self._expressions = []
        line = next(lines)
        while True:
            expr, lines = RulesFile.parse_line(chain([line], lines))
            if expr is not None:
                self._expressions.append(expr)
            try:
                line = next(lines)
            except StopIteration:
                return

    def transform(self, transaction: LedgerTransaction):
        for expr in self._expressions:
            transaction = expr.transform(transaction)
        return transaction

    @staticmethod
    def from_file(file_name: str):
        with open(file_name, 'r') as f:
            return RulesFile(f)

    @staticmethod
    def parse_line(lines: Iterator[str]):
        line = next(lines)
        tokens = line.split(maxsplit=1)
        if len(tokens) == 0:
            return None, lines

        if tokens[0] == 'if':
            expr = IfExpr()
        elif tokens[0] == 'account1' or tokens[0] == 'account2':
            expr = AccountExpr()
        elif tokens[0] == 'comment':
            expr = CommentExpr()
        else:
            raise Exception("Unsupported directive: " + tokens[0])

        lines = chain([line], lines)
        lines = expr.process(lines)
        return expr, lines

    def __str__(self):
        return "".join(map(str, self._expressions))


class IfExpr(object):
    def __init__(self):
        self._conditions = []
        self._exprs = []

    def process(self, lines):
        line = next(lines)
        if_stmt = line.split(maxsplit=1)
        if len(if_stmt) == 0 or if_stmt[0] != 'if':
            raise Exception("If statement doesn't start with if\n  " + line)
        if len(if_stmt) > 1:
            line = if_stmt[1]
        else:
            line = next(lines)
        while not line.startswith(' '):
            regex = re.compile(line.rstrip(), flags=re.IGNORECASE)
            self._conditions.append(regex)
            try:
                line = next(lines)
            except StopIteration:
                raise Exception("If statement does not have any expressions")
        if len(self._conditions) == 0:
            raise Exception("If statement doesn't have any conditions")
        while line.startswith(' '):
            expr, lines = RulesFile.parse_line(chain([line.lstrip()], lines))
            self._exprs.append(expr)
            try:
                line = next(lines)
            except StopIteration:
                if len(self._exprs) == 0:
                    raise Exception("If statement doesn't have any "
                                    "expressions")
                return lines
        if len(self._exprs) == 0:
            raise Exception("If statement doesn't have any expressions")
        return chain([line], lines)

    def transform(self, transaction: LedgerTransaction):
        # Simulate a line in a CSV during a real CSV import.
        first_txn = transaction.postings[0]
        transaction_str = ','.join([
            transaction.date_str(),
            '"%s"' % transaction.payee.replace('"', '\\"'),
            '$',
            str(first_txn.amount),
            str(first_txn.balance),
        ])
        for condition in self._conditions:
            if condition.search(transaction_str):
                for expr in self._exprs:
                    transaction = expr.transform(transaction)
                return transaction
        return transaction

    def __str__(self):
        lines = ["if"]
        lines += map(lambda p: p.pattern, self._conditions)
        lines += map(lambda e: "  %s" % e, self._exprs)
        return "\n".join(lines) + "\n"


class AccountExpr(object):
    def process(self, lines):
        line = next(lines)
        tokens = line.split(maxsplit=1)
        if len(tokens) < 2 or not tokens[0].startswith('account'):
            raise Exception("Account line must be of the form "
                            "'accountN ACCOUNT_NAME'\n  " + line)
        self._account_num = tokens[0][len('account'):]
        self._account_name = tokens[1].rstrip()
        return lines

    def transform(self, transaction: LedgerTransaction):
        if self._account_num == 1:
            transaction.postings[0].account = self._account_name
        else:
            transaction.postings[1].account = self._account_name
        return transaction

    def __str__(self):
        return "account{} {}\n".format(self._account_num, self._account_name)


class CommentExpr(object):
    def process(self, lines):
        line = next(lines)
        tokens = line.split(maxsplit=1)
        if len(tokens) < 2 or tokens[0] != 'comment':
            raise Exception("Comment line must be of the form "
                            "'comment [%comment] [text]'\n  " + line)
        self._comment = tokens[1].rstrip()
        return lines

    def transform(self, transaction: LedgerTransaction):
        comment = self._comment
        if transaction.comment is None:
            transaction.comment = ''
        if '%comment' in self._comment:
            comment = comment.replace('%comment', transaction.comment)
        transaction.comment = comment
        return transaction

import 'react-bootstrap-table-next/dist/react-bootstrap-table2.min.css';
import './Transactions.css';

import React from 'react';
import axios from 'axios';
import BootstrapTable from 'react-bootstrap-table-next';
import paginationFactory from 'react-bootstrap-table2-paginator';
import Amount from './Amount';
import Table from 'react-bootstrap/Table';
import Form from 'react-bootstrap/Form';
import { cleanCategory, CategoryPicker } from './Categories';


const columns = [
  {
    dataField: 'Date',
    text: 'Date',
    formatter: date => new Date(date).toDateString(),
    classes: 'transactions-no-wrap',
  },
  {
    dataField: 'Payee',
    text: 'Payee',
    headerClasses: 'transactions-large-width',
  },
  {
    dataField: 'Postings',
    text: 'Categories',
    formatter: postings => {
      let categories = postings.slice(1).map(p => cleanCategory(p.Account))
      let className = "category"
      if (categories.includes("uncategorized")) {
        className += " uncategorized"
      }
      return <span className={className}>{categories.join(", ")}</span>
    },
  },
  {
    dataField: 'SummaryAmount',
    text: 'Amount',
    align: 'right',
    headerAlign: 'right',
    formatter: (amount, txn) => {
      return <Amount amount={amount} prefix={txn.SummaryCurrency} />
    },
  },
];

function prepTransactions(transactions) {
  if (! transactions) {
    return []
  }
  transactions = transactions.map(t => {
    let id = t.Tags && t.Tags.id
    for (let i = 0; !id && i < t.Postings.length; i++) {
      id = t.Postings[i].Tags && t.Postings[i].Tags.id
    }
    return Object.assign({}, t, {
      ID: id,
      SummaryAmount: Number(t.Postings[0].Amount),
      SummaryCurrency: t.Postings[0].Currency,
      Postings: t.Postings.map(p =>
        Object.assign({}, p, {
          Amount: Number(p.Amount)
        })
      )
    })
  }).reverse()
  return transactions
}


export default function Transactions(props) {
  const [transactions, setTransactions] = React.useState([])
  const [count, setCount] = React.useState(1)
  const [page, setPage] = React.useState(1)

  const handleTableChange = (_, { page, sizePerPage = 10 }) => {
    axios.get('/api/v1/transactions', {
        params: { page, results: sizePerPage },
      })
      .then(res => {
        if (res.status !== 200 ) {
          throw new Error("Error fetching transactions")
        }
        let transactions = prepTransactions(res.data.Transactions)
        setTransactions(transactions)
        setCount(res.data.Count)
        setPage(page)
      })
  }

  const { syncTime } = props;
  React.useEffect(() => {
    handleTableChange(null, { page })
  }, [page, syncTime])

  const updateTransaction = txn => {
    let newTransactions = Array.from(transactions)
    let txnIndex = newTransactions.findIndex(t => t.ID === txn.ID)
    if (txnIndex === -1) {
      throw Error(`Tried to update invalid transaction: ${txn}`)
    }
    let { Postings } = txn
    axios.patch(`/api/v1/transactions/${txn.ID}`, { Postings })
      .then(res => {
        if (res.status !== 204 ) {
          throw new Error("Error updating transaction")
        }
        newTransactions[txnIndex] = Object.assign({}, newTransactions[txnIndex], txn)
        setTransactions(newTransactions)
      })
  }

  return (
    <div className="transactions">
      <BootstrapTable
        bootstrap4
        bordered={false}
        columns={ columns }
        data={ transactions }
        expandRow={{ renderer: transactionRow(updateTransaction) }}
        keyField='ID'
        onTableChange={ handleTableChange }
        pagination={ paginationFactory({
          page: page,
          totalSize: count,
        }) }
        remote
        wrapperClasses='table-responsive'
        />
    </div>
  )
}

function transactionRow(updateTransaction) {
  return txn => {
    let postings = txn.Postings.map((p, i) => {
      if (i === 0) {
        let account = p.Account;
        let separatorIndex = account.indexOf(':')
        account = separatorIndex !== -1 ? account.slice(separatorIndex + 1) : account
        account = account.replace(/:/, " - ")
        return Object.assign({}, p, { Account: account })
      }
      return p
    })

    const updatePosting = (index, newPosting) => {
      let { ID, Postings } = txn
      Postings = Array.from(Postings)
      Postings[index] = Object.assign({}, Postings[index], newPosting)
      updateTransaction({ ID, Postings })
    }

    return (
      <Form>
        <Table className="postings" responsive borderless>
          <tbody>
            {postings.map((posting, i) =>
              <tr key={posting.Account}>
                <td>
                  { i === 0
                    ? <Form.Control type="text" value={posting.Account} disabled />
                    : <CategoryPicker category={posting.Account} setCategory={c => updatePosting(i, { Account: c })} />
                  }
                </td>
                <td>
                  <Amount
                    amount={posting.Amount}
                    disabled={i === 0 || postings.length === 2}
                    editable
                    onChange={a => updatePosting(i, { Amount: a })}
                    prefix={posting.Currency}
                    />
                </td>
              </tr>
            )}
          </tbody>
        </Table>
      </Form>
    )
  }
}
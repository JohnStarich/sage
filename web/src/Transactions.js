import 'react-bootstrap-table-next/dist/react-bootstrap-table2.min.css';
import './Transactions.css';

import React from 'react';
import axios from 'axios';
import BootstrapTable from 'react-bootstrap-table-next';
import paginationFactory from 'react-bootstrap-table2-paginator';
import Amount from './Amount';


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
    dataField: 'Categories',
    text: 'Categories',
    classes: 'transactions-categories',
  },
  {
    dataField: 'Amount',
    text: 'Amount',
    align: 'right',
    headerAlign: 'right',
    formatter: (amount, txn) => {
      return <Amount amount={Number(amount)} prefix={txn.Currency} />
    },
  },
];

const cleanCategory = account => {
  let i = account.lastIndexOf(":")
  if (i === -1) {
    return account
  }
  return account.slice(i+1)
}

export default class Transactions extends React.Component {
  state = {
    transactions: [],
  }

  setTransactions(transactions) {
    if (! transactions) {
      this.setState({ transactions: [] });
      return
    }
    transactions = transactions.map(t => {
      let id = t.Tags && t.Tags.id
      for (let i = 0; !id && i < t.Postings.length; i++) {
        id = t.Postings[i].Tags && t.Postings[i].Tags.id
      }
      return {
        ID: id,
        Date: t.Date,
        Payee: t.Payee,
        Amount: t.Postings[0].Amount,
        Currency: t.Postings[0].Currency,
        Categories: t.Postings.slice(1)
                      .map(p => cleanCategory(p.Account))
                      .join(", "),
      }
    }).reverse()
    this.setState({ transactions })
  }

  componentDidMount() {
    this.handleTableChange(null, { page: 1 })
  }

  handleTableChange = (_, { page, sizePerPage = 10 }) => {
    axios.get('/api/v1/transactions', {
        params: { page, results: sizePerPage },
      })
      .then(res => {
        if (res.status !== 200 ) {
          throw new Error("Error fetching transactions")
        }
        this.setTransactions(res.data.Transactions)
        this.setState({ count: res.data.Count })
      })
  }

  render() {
    return (
      <div className="transactions">
        <BootstrapTable
          bootstrap4
          bordered={false}
          wrapperClasses='table-responsive'
          columns={ columns }
          data={ this.state.transactions }
          keyField='ID'
          onTableChange={ this.handleTableChange }
          pagination={ paginationFactory({
            page: this.state.page,
            totalSize: this.state.count,
          }) }
          remote
          />
      </div>
    )
  }
}

import 'react-bootstrap-table-next/dist/react-bootstrap-table2.min.css';
import './Transactions.css';

import API from './API';
import Amount from './Amount';
import BootstrapTable from 'react-bootstrap-table-next';
import Modal from 'react-bootstrap/Modal';
import React from 'react';
import RuleEditor from './RuleEditor';
import ToolkitProvider, { Search } from 'react-bootstrap-table2-toolkit';
import Transaction from './Transaction';
import paginationFactory from 'react-bootstrap-table2-paginator';
import { cleanCategory } from './CategoryPicker';


const dateFormatter = new Intl.DateTimeFormat('default', { year: 'numeric', month: 'numeric', day: 'numeric', timeZone: 'UTC' })

const columns = [
  {
    dataField: 'Date',
    text: 'Date',
    formatter: date => dateFormatter.format(new Date(date)),
    classes: 'table-no-wrap transactions-date',
  },
  {
    dataField: 'Payee',
    text: 'Payee',
    headerClasses: 'transactions-large-width',
    classes: 'table-hide-no-wrap',
  },
  {
    dataField: 'Postings',
    text: 'Categories',
    formatter: postings => {
      let categories = postings.slice(1).map(p => cleanCategory(p.Account))
      let className = null
      if (categories.includes("uncategorized")) {
        className = "uncategorized"
      }
      return <span className={className}>{categories.join(", ")}</span>
    },
    headerClasses: 'table-hide-no-wrap',
    classes: "category-name table-hide-no-wrap",
  },
  {
    dataField: 'SummaryAmount',
    text: 'Amount',
    align: 'right',
    headerAlign: 'right',
    formatter: (amount, txn) =>
      <Amount className="transaction amount-finance" amount={amount} prefix={txn.SummaryCurrency} />,
  },
];

function prepTransactions(transactions) {
  if (!transactions) {
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
  const [search, setSearch] = React.useState("")
  const [accountIDMap, setAccountIDMap] = React.useState(null)
  const [activeRule, setActiveRule] = React.useState(null)

  const handleTableChange = (_, { page, sizePerPage = 10, searchText = search }) => {
    if (search !== searchText) {
      page = 1
      setPage(1)
      setSearch(searchText)
    }
    API.get('/v1/getTransactions', {
      params: { page, results: sizePerPage, search: searchText },
    })
      .then(res => {
        let transactions = prepTransactions(res.data.Transactions)
        setTransactions(transactions)
        setCount(res.data.Count)
        setPage(page)
        setAccountIDMap(res.data.AccountIDMap)
      })
  }

  const { syncTime } = props;
  React.useEffect(() => {
    handleTableChange(null, { page })
  }, [syncTime]) // eslint-disable-line react-hooks/exhaustive-deps

  const updateTransaction = txn => {
    let newTransactions = Array.from(transactions)
    let txnIndex = newTransactions.findIndex(t => t.ID === txn.ID)
    if (txnIndex === -1) {
      throw Error(`Tried to update invalid transaction: ${txn}`)
    }
    let { Postings } = txn
    API.post('/v1/updateTransaction', { ID: txn.ID, Postings })
      .then(res => {
        newTransactions[txnIndex] = Object.assign({}, newTransactions[txnIndex], txn)
        setTransactions(newTransactions)
      })
  }

  const editRule = (txn, posting) => {
    setActiveRule({ transaction: txn }) // TODO fetch existing rule details
  }

  return (
    <div className="transactions">
      <ToolkitProvider
        keyField="ID"
        data={transactions}
        columns={columns}
        search
      >
        {toolkitprops =>
          <div key="0">
            <Search.SearchBar
              {...toolkitprops.searchProps}
              delay={1000}
              className="search"
              tabIndex="0"
            />
            <BootstrapTable
              {...toolkitprops.baseProps}
              bootstrap4
              bordered={false}
              expandRow={{ renderer: Transaction(updateTransaction, accountIDMap, editRule) }}
              noDataIndication="No transactions found"
              onTableChange={handleTableChange}
              pagination={paginationFactory({
                page: page,
                totalSize: count,
              })}
              remote
              wrapperClasses='table-responsive'
            />
          </div>
        }
      </ToolkitProvider>
      <Modal show={activeRule !== null}>
        <RuleEditor onClose={() => setActiveRule(null)} {...activeRule} />
      </Modal>
    </div>
  )
}

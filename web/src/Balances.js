import React from 'react';
import axios from 'axios';
import './Balances.css';
import Amount from './Amount';
import Table from 'react-bootstrap/Table';
import Button from 'react-bootstrap/Button';
import Popover from 'react-bootstrap/Popover';
import OverlayTrigger from 'react-bootstrap/OverlayTrigger';
import { Link } from 'react-router-dom';


export default function Balances({ syncTime }) {
  const [accounts, setAccounts] = React.useState(null)
  const [messages, setMessages] = React.useState([])

  React.useEffect(() => {
    axios.get('/api/v1/getBalances')
      .then(res => {
        setAccounts(res.data.Accounts || [])
        setMessages(res.data.Messages || [])
      })
  }, [syncTime])

  if (accounts === null) {
    return <div className="balances"><em>Loading...</em></div>
  }

  const messageMap = messages.reduce((acc, message) => {
    if (!message.AccountID) {
      return acc
    }
    let id = message.AccountID
    if (!acc[id]) {
      acc[id] = []
    }
    acc[id].push(message)
    return acc
  }, {})

  const accountIDs = new Set(accounts.map(a => a.ID))

  const nonAccountMessages =
    messages
      .filter(m => !m.AccountID)
      .map(m => m.Message)
  const remainingAccountMessages =
    Object.keys(messageMap)
      .filter(id => !accountIDs.has(id))
      .map(id => messageMap[id])

  const accountTypes = accounts.reduce((acc, account) => {
    if (account.Balances) {
      if (acc[account.AccountType] === undefined) {
        acc[account.AccountType] = []
      }
      acc[account.AccountType].push(account)
    }
    return acc
  }, {})

  const balanceSections =
    Object.entries(accountTypes)
      .map(([type, accounts]) => <BalanceSection key={type} name={type} accounts={accounts} getMessages={account => messageMap[account.ID]} />)

  return (
    <div className="balances">
          {accounts.length === 0 ?
            <tr>
              <td><Link to="/accounts" className="btn btn-outline-primary">Add your first account</Link></td>
              <td></td>
            </tr>
          : balanceSections}
          {remainingAccountMessages.map(msgs =>
            <p key={msgs[0].AccountID} className="message">
              <strong>{msgs[0].AccountName}</strong>
              <WarningTooltip messages={msgs.map(m => m.Message)} />
            </p>
          )}
          {nonAccountMessages.map((m, i) =>
            <p key={i}>{m}</p>
          )}
    </div>
  )
}

function WarningTooltip({ messages }) {
  const popover = (
    <Popover>
      <div className="balances-warning-overlay">
        <strong>{messages && messages.length > 1 ? `${messages.length} issues` : '1 issue'}</strong>
        <ul>
          {messages.map((m, i) =>
            <li key={i}>
              {m === "Missing opening balance"
                ? <Link to="/balances">{m}</Link>
                : m
              }
            </li>
          )}
        </ul>
      </div>
    </Popover>
  )

  return (
    <OverlayTrigger trigger="click" placement="left" overlay={popover}>
      <Button variant="warning">
        {messages && messages.length > 1 ? `${messages.length} issues` : '1 issue'}
      </Button>
    </OverlayTrigger>
  );
}

function BalanceSection({ name, accounts, getMessages }) {
  const nameRemappings = {
    "assets": "Cash",
    "liabilities": "Debts",
  }
  if (nameRemappings[name]) {
    name = nameRemappings[name]
  }
  const total =
    accounts
      .map(a => Number(a.Balances[a.Balances.length - 1]))
      .reduce((a, b) => a + b)
  const balance = account => Number(account.Balances[account.Balances.length - 1])

  let headerClass = "amount-finance"
  if (total > 0 && name === "Cash") {
    headerClass += " balance-cash"
  }

  const renderMessages = account => {
    const messages = getMessages(account)
    if (messages) {
      return <td className="balance-warning"><WarningTooltip messages={messages.map(m => m.Message)} /></td>
    }
    return <td></td>
  }
  return (
    <Table responsive>
      <thead>
        <tr><th>{name}</th><th><Amount prefix="$" amount={total} className={headerClass} /></th></tr>
      </thead>
      <tbody>
        {accounts.map(account =>
          <tr key={account.ID}>
            <td>{account.Account}</td>
            <td>
              {account.Balances ? <Amount prefix="$" amount={balance(account)} className="amount-finance" /> : null}
            </td>
            {renderMessages(account)}
          </tr>
        )}
      </tbody>
    </Table>
  )
}

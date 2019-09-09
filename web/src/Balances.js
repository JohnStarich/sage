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

  return (
    <div className="balances">
      <Table responsive>
        <tbody>
          <tr><th>Account</th><th>Balance</th></tr>
          {accounts.length !== 0 ? null :
            <tr>
              <td><Link to="/accounts" className="btn btn-outline-primary">Add your first account</Link></td>
              <td></td>
            </tr>
          }
          {accounts.map(account =>
            <tr key={account.ID}>
              <td>{account.Account}</td>
              <td>
                {!account.Balances ? null :
                  <Amount
                    amount={Number(account.Balances[account.Balances.length - 1])}
                    prefix="$"
                    className="amount-finance"
                  />
                }
              </td>
              {!messageMap[account.ID] ? <td></td> :
                <td className="balance-warning"><WarningTooltip messages={messageMap[account.ID].map(m => m.Message)} /></td>
              }
            </tr>
          )}
          {remainingAccountMessages.map(msgs =>
            <tr key={msgs[0].AccountID} className="message">
              <td>{msgs[0].AccountName}</td>
              <td></td>
              <td className="balance-warning"><WarningTooltip messages={msgs.map(m => m.Message)} /></td>
            </tr>
          )}
          {nonAccountMessages.map((m, i) =>
            <tr key={i}><td colSpan="3">{m}</td></tr>
          )}
        </tbody>
      </Table>
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

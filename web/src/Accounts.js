import React from 'react';
import { Route, Link } from "react-router-dom";
import axios from 'axios';

import Account from './Account';
import Crumb from './Breadcrumb';

export default function Accounts({ match }) {
  const [accounts, setAccounts] = React.useState([])
  React.useEffect(() => {
    axios.get('/api/v1/accounts')
      .then(res => setAccounts(res.data.Accounts))
  }, [])

  const accountUpdated = (originalAccountID, account) => {
    let newAccounts = Array.from(accounts)
    for (let i in newAccounts) {
      if (newAccounts[i].ID === originalAccountID) {
        newAccounts[i] = account
      }
    }
    setAccounts(newAccounts)
  }

  return (
    <>
      <Crumb title="Accounts" match={match} />
      <Route exact path={match.path} render={() =>
        <ul className="accounts">
          {accounts.map(a =>
            <li key={a.ID}><Link to={`${match.url}/${a.ID}`}>{a.Description}</Link></li>
          )}
        </ul>
      } />
      <Route path={`${match.path}/:id`} component={props => <AccountView updated={accountUpdated} {...props} />} />
    </>
  )
}

function AccountView({ updated, match }) {
  const [account, setAccount] = React.useState(null)
  React.useEffect(() => {
    axios.get(`/api/v1/accounts/${match.params.id}`)
      .then(res => {
        setAccount(res.data.Account)
      })
  }, [match.params.id])

  return (
    <>
      <Crumb title={account ? account.Description : 'Loading...'} match={match} />
      <Account account={account} editable updated={updated} />
    </>
  )
}

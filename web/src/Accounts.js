import React from 'react';
import { Route, Link } from "react-router-dom";
import axios from 'axios';

import Account from './Account';
import Crumb from './Breadcrumb';

export default function Accounts({ match }) {
  const [accounts, setAccounts] = React.useState([])
  React.useEffect(() => {
    axios.get('/api/v1/accounts')
      .then(res => {
        if (res.status !== 200 ) {
          throw new Error("Error fetching accounts")
        }
        setAccounts(res.data.Accounts)
      })
  }, [])

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
      <Route path={`${match.path}/:id`} component={AccountView} />
    </>
  )
}

function AccountView({ match }) {
  const [account, setAccount] = React.useState(null)
  React.useEffect(() => {
    axios.get(`/api/v1/accounts/${match.params.id}`)
      .then(res => {
        if (res.status !== 200 ) {
          throw new Error("Error fetching account with ID " + match.params.id)
        }
        setAccount(res.data.Account)
      })
  }, [match.params.id])
  return (
    <>
      <Crumb title={account ? account.Description : 'Loading...'} match={match} />
      <Account account={account} editable />
    </>
  )
}

import React from 'react';
import { Route, Link } from "react-router-dom";
import axios from 'axios';
import './Accounts.css';

import Account from './Account';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Crumb from './Breadcrumb';
import Row from 'react-bootstrap/Row';

export default function Accounts({ match }) {
  const [accounts, setAccounts] = React.useState([])
  React.useEffect(() => {
    axios.get('/api/v1/accounts')
      .then(res => setAccounts(res.data.Accounts))
  }, [])

  const accountCreated = account => {
    setAccounts([account].concat(accounts))
  }

  const accountUpdated = (originalAccountID, account) => {
    let newAccounts = Array.from(accounts)
    for (let i in newAccounts) {
      if (newAccounts[i].ID === originalAccountID) {
        newAccounts[i] = account
      }
    }
    setAccounts(newAccounts)
  }

  const deleteAccount = accountID => {
    let account = accounts.find(a => a.ID === accountID)
    if (!window.confirm(`Are you sure you want to delete your account '${account.Description}'?`)) {
      return
    }
    axios.delete(`/api/v1/accounts/${accountID}`)
      .then(() =>
        setAccounts(
          accounts.filter(a => a.ID !== accountID)))
      .catch(e => {
        if (e.response.status !== 400) {
          throw e
        }
        alert(e.response.data.Error)
      })
  }

  return (
    <>
      <Crumb title="Accounts" match={match} />
      <Route exact path={match.path} render={() =>
        <>
          <Container className="accounts">
            <Row>
              <Col><h2>Accounts</h2></Col>
            </Row>
            {accounts.map(a =>
              <Row key={a.ID}>
                <Col>{a.Description}</Col>
                <Col className="account-buttons">
                  <Link to={`${match.url}/edit/${a.ID}`} className="btn btn-outline-secondary">Edit</Link>
                  <Button variant="outline-danger" onClick={() => deleteAccount(a.ID)}>Delete</Button>
                </Col>
              </Row>
            )}
            <Row>
              <Col><Link to={`${match.url}/new`} className="btn btn-primary add-new">Add new</Link></Col>
            </Row>
          </Container>
        </>
      } />
      <Route path={`${match.path}/edit/:id`} component={props => <AccountEditor updated={accountUpdated} {...props} />} />
      <Route path={`${match.path}/new`} component={props => <NewAccount created={accountCreated} {...props} />} />
    </>
  )
}

function NewAccount({ created, match }) {
  const updated = (_, account) => {
    if (created) {
      created(account)
    }
  }
  return (
    <>
      <Crumb title="New" match={match} />
      <Account editable updated={updated} />
    </>
  )
}

function AccountEditor({ updated, match }) {
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

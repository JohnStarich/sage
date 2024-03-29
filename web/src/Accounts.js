import React from 'react';
import { Route, Link } from "react-router-dom";
import API from './API';
import './Accounts.css';

import CommonAccount from './CommonAccount';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Crumb from './Breadcrumb';
import DirectConnect from './DirectConnect';
import ImportAccounts from './ImportAccounts';
import Row from 'react-bootstrap/Row';
import WebConnect from './WebConnect';

export default function Accounts({ match }) {
  const [accounts, setAccounts] = React.useState([])
  React.useEffect(() => {
    API.get('/v1/getAccounts')
      .then(res => {
        if (res.data.Accounts) {
          setAccounts(res.data.Accounts)
        }
      })
  }, [])

  const accountCreated = (...newAccounts) => {
    setAccounts(newAccounts.concat(accounts))
  }

  const accountUpdated = (originalAccountID, account) => {
    let newAccounts = Array.from(accounts)
    for (let i in newAccounts) {
      if (newAccounts[i].AccountID === originalAccountID) {
        newAccounts[i] = account
      }
    }
    setAccounts(newAccounts)
  }

  const deleteAccount = accountID => {
    let account = accounts.find(a => a.AccountID === accountID)
    if (!window.confirm(`Are you sure you want to delete your account '${account.AccountDescription}'?`)) {
      return
    }
    API.get('/v1/deleteAccount', {
      params: { id: accountID },
    })
      .then(() =>
        setAccounts(
          accounts.filter(a => a.AccountID !== accountID)))
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
        <Container className="accounts">
          <Row>
            <Col><h2>Accounts</h2></Col>
          </Row>
          <Row>
            <Col>
              <p>Add a new account to automatically download transactions directly from your institution.</p>
              <p>Alternatively, import an OFX or QFX file downloaded from your institution.</p>
            </Col>
          </Row>
          {accounts.map(a =>
            <Row key={a.AccountID}>
              <Col>{a.AccountDescription}</Col>
              <Col className="account-buttons">
                <Link to={`${match.url}/edit/${a.AccountID}`} className="btn btn-outline-secondary">Edit</Link>
                <Button variant="outline-danger" onClick={() => deleteAccount(a.AccountID)}>Delete</Button>
              </Col>
            </Row>
          )}
          <Row>
            <Col className="account-actions">
              <Link to={`${match.url}/new`} className="btn btn-primary new">Add New</Link>
              <Link to={`${match.url}/import`} className="btn btn-secondary import">Import OFX/QFX</Link>
            </Col>
          </Row>
        </Container>
      } />
      <Route path={`${match.path}/edit/:id`} component={props => <AccountEditor updated={accountUpdated} {...props} />} />
      <Route path={`${match.path}/new`} component={props => <NewAccount created={accountCreated} {...props} />} />
      <Route path={`${match.path}/advanced-direct-connect`} component={props => <NewDirectConnect created={accountCreated} {...props} />} />
      <Route path={`${match.path}/import`} component={Import} />
    </>
  )
}

function NewAccount({ created, match }) {
  return (
    <>
      <Crumb title="New Account" match={match} />
      <Container>
        <Row><Col><h2>New Account</h2></Col></Row>
        <Row>
          <Col>
            <p><em>Advanced users can enter all direct connect details <Link to={`/settings/accounts/advanced-direct-connect`}>here</Link>.</em></p>
          </Col>
        </Row>
        <Row>
          <CommonAccount created={created} />
        </Row>
      </Container>
    </>
  )
}

function NewDirectConnect({ created, match }) {
  const updated = (_, account) => {
    if (created) {
      created(account)
    }
  }
  return (
    <>
      <Crumb title="Advanced Direct Connect" match={match} />
      <Container>
        <Row><Col><h2>Advanced Direct Connect</h2></Col></Row>
        <Row>
          <Col>
            <p>For advanced users only. Input known direct connect details to add an account.</p>
            <p>Sometimes the password is a PIN rather than the sign-in password, and the username could be an ID only provided in their instructions.</p>
            &nbsp;
            <p>If you're <strong>not</strong> an advanced user, then use the new account page <Link to="/settings/accounts/new">here</Link>.</p>
          </Col>
        </Row>
        <Row>
          <DirectConnect editable updated={updated} />
        </Row>
      </Container>
    </>
  )
}

function AccountEditor({ updated, match }) {
  const [account, setAccount] = React.useState(null)
  React.useEffect(() => {
    API.get('/v1/getAccount', {
      params: { id: match.params.id },
    })
      .then(res => {
        setAccount(res.data.Account)
      })
  }, [match.params.id])

  let Editor = DirectConnect
  if (account && account.WebConnect) {
    Editor = WebConnect
  }

  return (
    <>
      <Crumb title={account ? account.AccountDescription : 'Loading...'} match={match} />
      <Editor account={account} editable updated={updated} />
    </>
  )
}

function Import({ match }) {
  return (
    <>
      <Crumb title="Import" match={match} />
      <ImportAccounts />
    </>
  )
}

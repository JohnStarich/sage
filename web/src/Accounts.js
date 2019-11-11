import React from 'react';
import { Route, Link } from "react-router-dom";
import axios from 'axios';
import './Accounts.css';

import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Crumb from './Breadcrumb';
import DirectConnect from './DirectConnect';
import ExpressDirectConnect from './ExpressDirectConnect';
import ImportAccounts from './ImportAccounts';
import Row from 'react-bootstrap/Row';
import WebConnect from './WebConnect';

export default function Accounts({ match }) {
  const [accounts, setAccounts] = React.useState([])
  React.useEffect(() => {
    axios.get('/api/v1/getAccounts')
      .then(res => setAccounts(res.data.Accounts))
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
    axios.get('/api/v1/deleteAccount', {
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
        <>
          <Container className="accounts">
            <Row>
              <Col><h2>Accounts</h2></Col>
            </Row>
            <Row>
              <Col>
                <p>Add a new Direct Connect or Web Connect account to automatically download transactions directly from your institution.</p>
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
                <Link to={`${match.url}/direct-connect`} className="btn btn-primary add-new">Add new Direct Connect</Link>
                <Link to={`${match.url}/import`} className="btn btn-secondary import">Import OFX/QFX</Link>
                <br />
                <br />
                <Link to={`${match.url}/web-connect`} className="btn btn-warning add-new">Add new Web Connect <sup>(beta)</sup></Link>
              </Col>
            </Row>
          </Container>
        </>
      } />
      <Route path={`${match.path}/edit/:id`} component={props => <AccountEditor updated={accountUpdated} {...props} />} />
      <Route path={`${match.path}/direct-connect`} component={props => <ExpressDirectConnectAccounts created={accountCreated} {...props} />} />
      <Route path={`${match.path}/advanced-direct-connect`} component={props => <NewDirectConnect created={accountCreated} {...props} />} />
      <Route path={`${match.path}/web-connect`} component={props => <NewWebConnect created={accountCreated} {...props} />} />
      <Route path={`${match.path}/import`} component={Import} />
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
            <p>If you're <strong>not</strong> an advanced user, then use the express direct connect page <Link to="/accounts/direct-connect">here</Link>.</p>
          </Col>
        </Row>
        <Row>
          <DirectConnect editable updated={updated} />
        </Row>
      </Container>
    </>
  )
}

function NewWebConnect({ created, match }) {
  const updated = (_, account) => {
    if (created) {
      created(account)
    }
  }
  return (
    <>
      <Crumb title="Web Connect Beta" match={match} />
      <Container>
        <Row><Col><h2>Web Connect <sup>(beta)</sup></h2></Col></Row>
        <Row>
          <WebConnect editable updated={updated} />
        </Row>
      </Container>
    </>
  )
}

function ExpressDirectConnectAccounts({ created, match }) {
  return (
    <>
      <Crumb title="Direct Connect" match={match} />
      <ExpressDirectConnect created={created} />
    </>
  )
}

function AccountEditor({ updated, match }) {
  const [account, setAccount] = React.useState(null)
  React.useEffect(() => {
    axios.get('/api/v1/getAccount', {
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

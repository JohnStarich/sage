import './ExpressDirectConnect.css';
import API from './API';
import Accordion from 'react-bootstrap/Accordion';
import Button from 'react-bootstrap/Button';
import Card from 'react-bootstrap/Card';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import LoadingButton from './LoadingButton';
import Password from './Password';
import React from 'react';
import Row from 'react-bootstrap/Row';
import { Link } from "react-router-dom";


const labelWidth = 4
const inputWidth = 12 - labelWidth

export default function({ driver, created }) {
  const [validated, setValidated] = React.useState(false)
  const [accounts, setAccounts] = React.useState(null)
  const [findFeedback, setFindFeedback] = React.useState(null)
  const [findResult, setFindResult] = React.useState(null)
  const [addFeedback, setAddFeedback] = React.useState(null)
  const [submittingAccounts, setSubmittingAccounts] = React.useState(false)

  if (! created) {
    throw Error("Created prop must be set")
  }

  const valueFromID = id => document.getElementById(id).value
  const submit = async e => {
    e.preventDefault()
    e.stopPropagation()
    setValidated(true)
    const form = e.currentTarget
    if (form.checkValidity() === false) {
      return
    }
    setFindFeedback(null)
    setFindResult(null)
    const password = valueFromID("password")
    const res = await API.post('/v1/direct/fetchAccounts', {
      InstDescription: driver.Description,
      InstFID: driver.FID,
      InstOrg: driver.Org,
      ConnectorURL: driver.URL,
      ConnectorUsername: valueFromID("username"),
      ConnectorPassword: password,
      ConnectorConfig: {
        ClientID: valueFromID("clientID"),
        AppID: valueFromID("appID"),
        AppVersion: valueFromID("appVersion"),
        OFXVersion: valueFromID("ofxVersion"),
      },
    })

    try {
      if (! res.data || res.data.length === 0) {
        setFindFeedback("No accounts found")
        return
      }
      res.data.forEach(a => {
        a.DirectConnect.ConnectorPassword = password // copy in password since API redacts it
      })
      setAccounts(res.data)
      setFindResult("Success! Select desired accounts below:")
    } catch(e) {
      if (!e.response || !e.response.data || !e.response.data.Error) {
        console.error(e)
        throw e
      }
      setFindFeedback(e.response.data.Error)
    }
  }

  return (
    <Container>
      <Form
        noValidate
        validated={validated}
        onSubmit={submit}
      >
        <Form.Group controlId="username" as={Row}>
          <Form.Label column sm={labelWidth}>Username</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control type="text" required autoFocus />
          </Col>
        </Form.Group>
        <Form.Group controlId="password" as={Row}>
          <Form.Label column sm={labelWidth}>Password</Form.Label>
          <Col sm={inputWidth}>
            <Password />
          </Col>
        </Form.Group>

        <Form.Group>
          <Accordion>
            <Card>
              <Card.Header>
                <Accordion.Toggle as={Button} variant="link" eventKey="0">
                  Advanced Client Options
                </Accordion.Toggle>
              </Card.Header>
              <Accordion.Collapse eventKey="0">
                <Card.Body>
                  <p>If you're an advanced user, and have all of your connection details already, then enter them manually <Link to="/accounts/advanced-direct-connect">here</Link>.</p>
                  <Form.Group controlId="clientID" as={Row}>
                    <Form.Label column sm={labelWidth}>Client ID</Form.Label>
                    <Col sm={inputWidth}>
                      <Form.Control type="text" placeholder="Optional" />
                    </Col>
                  </Form.Group>

                  <Form.Group controlId="appID" as={Row}>
                    <Form.Label column sm={labelWidth}>Client App ID</Form.Label>
                    <Col sm={inputWidth}>
                      <Form.Control type="text" defaultValue="QWIN" placeholder="QWIN" required />
                    </Col>
                  </Form.Group>

                  <Form.Group controlId="appVersion" as={Row}>
                    <Form.Label column sm={labelWidth}>Client Version</Form.Label>
                    <Col sm={inputWidth}>
                      <Form.Control type="text" defaultValue="2500" placeholder="2500" required />
                    </Col>
                  </Form.Group>

                  <Form.Group controlId="ofxVersion" as={Row}>
                    <Form.Label column sm={labelWidth}>OFX Version</Form.Label>
                    <Col sm={inputWidth}>
                      <Form.Control type="text" defaultValue="102" placeholder="102" required />
                    </Col>
                  </Form.Group>
                </Card.Body>
              </Accordion.Collapse>
            </Card>
          </Accordion>
        </Form.Group>
        &nbsp;
        <Form.Row>
          <Col sm={labelWidth}>
            <LoadingButton type="submit" onClick={submit}>Find</LoadingButton>
          </Col>
          <Col>
            <p>{findFeedback}</p>
            <p>{findResult}</p>
          </Col>
        </Form.Row>
      </Form>
      &nbsp;

      {findFeedback && !(findResult && findResult.startsWith("Success! ")) ?
        <p>If your username and password don't work, visit your institution's website for more information on "Direct Connect." Instructions for Direct Connect may be located under "QuickBooks" or "Quicken." Sometimes the username is an account ID and the password a PIN instead.</p>
      : null}

      {!accounts ? null :
        <Form
          noValidate
          onSubmit={e => {
            e.preventDefault()
            e.stopPropagation()
            const form = e.currentTarget
            if (form.checkValidity() !== false) {
              setAddFeedback(null)
              setSubmittingAccounts(true)
              Promise.all(accounts.map(async account => {
                const checkbox = document.getElementById("add-account-id-" + account.AccountID)
                const accountName = document.getElementById("add-account-name-" + account.AccountID).value
                if (!checkbox.checked || checkbox.disabled) {
                  return null
                }
                const updatedAccount = Object.assign({}, account, { AccountDescription: accountName })
                await API.post('/v1/addAccount', updatedAccount)
                checkbox.disabled = true
                checkbox.classList.add("is-valid")
                return updatedAccount
              }))
                .then(accounts => {
                  setTimeout(() => {
                    document.getElementById("return-to-accounts").click()
                    created(...accounts)
                  }, 1000)
                })
                .catch(e => {
                  if (!e.response || !e.response.data || !e.response.data.Error) {
                    console.error(e)
                    throw e
                  }
                  setAddFeedback(e.response.data.Error)
                })
                .finally(() => {
                  setSubmittingAccounts(false)
                })
            }
          }}
        >
          {accounts.map(a =>
            <Form.Row key={a.AccountID} className="account-suggestion">
              <Form.Group controlId={"add-account-id-" + a.AccountID} as={Col} sm="5">
                <Form.Check type="checkbox" label={a.AccountDescription} readOnly={submittingAccounts} />
              </Form.Group>
              <Form.Group controlId={"add-account-name-" + a.AccountID} as={Col} sm="7">
                <Form.Control type="text" defaultValue={
                  a.AccountDescription.includes(' ')
                  ? a.AccountDescription
                  : `${a.DirectConnect.InstDescription} - ****${a.AccountDescription.substring(a.AccountDescription.length - 4)}`
                } />
              </Form.Group>
            </Form.Row>
          )}
          <Form.Row>
            <Col sm={labelWidth}>
              <Button
                variant="outline-secondary"
                disabled={submittingAccounts}
                onClick={() => {
                  accounts.forEach(a => {
                    document.getElementById("add-account-id-"+a.AccountID).checked = true
                  })
                }}
              >
                Select All
              </Button>
            </Col>
          </Form.Row>
          &nbsp;
          <Form.Row>
            <Col sm={labelWidth}><Button type="submit" disabled={submittingAccounts}>Add Selected</Button></Col>
            <Col>{addFeedback}</Col>
          </Form.Row>
        </Form>
      }
      <Link id="return-to-accounts" to="/accounts" style={{display:"none"}}>Back to accounts</Link>
    </Container>
  )
}

import React from 'react';
import API from './API';
import './WebConnect.css';

import RadioGroup from './RadioGroup';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import Password from './Password';
import Row from 'react-bootstrap/Row';
import { Redirect } from 'react-router-dom';


let driverNames = null

function getDriverNames() {
  if (driverNames === null) {
    driverNames = API.get('/v1/web/getDriverNames')
      .then(res => res.data.DriverNames)
  }
  return driverNames
}

export default function ({ account, editable, updated }) {
  const [validated, setValidated] = React.useState(false)
  const [redirect, setRedirect] = React.useState(null)
  const [feedback, setFeedback] = React.useState(null)
  const [driverName, setDriverName] = React.useState(null)
  const [drivers, setDrivers] = React.useState(null)

  React.useEffect(() => {
    getDriverNames().then(drivers => {
      setDrivers(drivers)
      if (account) {
        const driver = drivers.find(d => d.toLowerCase() === account.WebConnect.Driver)
        setDriverName(driver || account.WebConnect.Driver)
      }
    })
  }, [account])

  if (account === null) {
    // prop was defined but hasn't loaded
    return null
  }

  const id = account ? account.AccountID : 'new'

  const labelWidth = 4
  const inputWidth = 12 - labelWidth
  const makeID = formIDFactory(id)

  const formControlDefaults = {
    disabled: !editable,
    onBlur: e => {
      e.target.classList.remove('is-valid', 'is-invalid')
      if (e.target.checkValidity() === false) {
        e.target.classList.add('is-invalid')
      } else {
        e.target.classList.add('is-valid')
      }
    },
  }

  return (
    <Container className="web-connect">
      {redirect}
      <Form
        id={makeID("form")}
        noValidate
        validated={validated}
        onSubmit={e => {
          e.preventDefault()
          e.stopPropagation()
          const form = e.currentTarget
          if (form.checkValidity() !== false) {
            const newAccount = accountFromForm(id)
            updateAccount(account ? account.AccountID : null, newAccount)
              .then(() => {
                setRedirect(<Redirect to="/accounts" />)
                if (updated) {
                  updated(id, newAccount)
                }
              })
              .catch(err => {
                setFeedback((err.response.data && err.response.data.Error) || "An internal server error occurred")
                if (!err.response.data || !err.response.data.Error) {
                  throw err
                }
              })
          }
          setValidated(true)
        }}
      >
        <Form.Group controlId={makeID("driver")} as={Row}>
          <Form.Label column sm={labelWidth}>Institution</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control
              as="select"
              onChange={e => setDriverName(e.target.value)}
              value={driverName || ""}
              required
              >
              <option value="">Select an institution...</option>
              {drivers && drivers.map((o, i) =>
                <option key={i}>{o}</option>
              )}
            </Form.Control>
          </Col>
        </Form.Group>

        {driverName ?
          <>
            <Row>
              <Form.Label column sm={labelWidth}>Account Description</Form.Label>
              <Col><Form.Control id={makeID("description")} type="text" defaultValue={account ? account.AccountDescription : driverName} {...formControlDefaults} required /></Col>
            </Row>
            <Form.Group controlId={makeID("id")} as={Row}>
              <Form.Label column sm={labelWidth}>Account Number</Form.Label>
              <Col sm={inputWidth}>
                <Form.Control type="text" defaultValue={account ? account.AccountID : null} {...formControlDefaults} autoFocus required />
              </Col>
            </Form.Group>
            <RadioGroup
              name={makeID("type")}
              choices={['Credit Card', 'Bank']}
              defaultChoice={account && account.AccountType === 'assets' ? 'Bank' : 'Credit Card'}
              label="Type of account?"
              smColumns={[labelWidth, inputWidth]}
            />
            <Form.Group>
              <Form.Group controlId={makeID("username")} as={Row}>
                <Form.Label column sm={labelWidth}>Username</Form.Label>
                <Col sm={inputWidth}>
                  <Form.Control
                    type="text"
                    defaultValue={account && account.WebConnect && account.WebConnect.Data.PasswordConnector ? account.WebConnect.Data.PasswordConnector.ConnectorUsername : null}
                    {...formControlDefaults}
                    required
                    />
                  <Form.Control.Feedback type="invalid">
                    Please choose a username.
                  </Form.Control.Feedback>
                </Col>
              </Form.Group>

              <Form.Group controlId={makeID("password")} as={Row}>
                <Form.Label column sm={labelWidth}>Password</Form.Label>
                <Col sm={inputWidth}>
                  <Password
                    required={!account}
                    {...formControlDefaults}
                  />
                  <Form.Control.Feedback type="invalid">
                    <p>A password is required when adding a new account</p>
                  </Form.Control.Feedback>
                </Col>
              </Form.Group>
            </Form.Group>
          </>
          : null}
        {!feedback ? null :
          <Row>
            <Col className="web-connect-test-failed">
              {feedback.trim().split("\n").map(line =>
                <span key={line}>{line}<br /></span>
              )}
            </Col>
          </Row>
        }
        <Form.Row>
          <Col><Button type="submit">{account ? 'Save' : 'Add'}</Button></Col>
        </Form.Row>
      </Form>
    </Container>
  )
}

function formIDFactory(accountID) {
  return name => `web-connect-${accountID}-${name}`
}

function accountFromForm(originalAccountID) {
  const makeID = formIDFactory(originalAccountID)
  const valueFromID = name => {
    const elem = document.getElementById(makeID(name))
    return elem ? elem.value : null
  }
  const valueFromName = name => {
    const elems = document.getElementsByName(makeID(name))
    for (let elem of elems) {
      if (elem.checked) {
        return elem.value
      }
    }
    return null
  }
  const formAccountType = valueFromName("type")
  let accountType;
  if (formAccountType === 'Bank') {
    accountType = 'assets'
  } else if (formAccountType === 'Credit Card') {
    accountType = 'liabilities'
  } else {
    throw Error("Invalid account type in form: "+formAccountType)
  }
  return {
    AccountID: valueFromID("id"),
    AccountDescription: valueFromID("description"),
    AccountType: accountType,
    WebConnect: {
      Driver: valueFromID("driver").toLowerCase(),
      Data: {
        PasswordConnector: {
          ConnectorUsername: valueFromID("username"),
          ConnectorPassword: valueFromID("password"),
        },
      },
    },
  }
}

function updateAccount(originalAccountID, account) {
  if (originalAccountID) {
    return API.post('/v1/updateAccount', Object.assign({}, { AccountID: originalAccountID }, account))
  }
  return API.post('/v1/addAccount', account)
}

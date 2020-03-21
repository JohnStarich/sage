import React from 'react';
import API from './API';

import './ExpressWebConnect.css';
import RadioGroup from './RadioGroup';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import Password from './Password';
import Row from 'react-bootstrap/Row';
import { Redirect } from 'react-router-dom';


export default function ({ driver, created }) {
  const [validated, setValidated] = React.useState(false)
  const [redirect, setRedirect] = React.useState(null)
  const [feedback, setFeedback] = React.useState(null)
  const [isBank, setIsBank] = React.useState(null)

  if (isBank === null) {
    // TODO add better credit card / bank detection
    setIsBank(!driver.Description.toLowerCase().includes('card'))
    return null
  }

  const labelWidth = 4
  const inputWidth = 12 - labelWidth
  const makeID = formIDFactory()

  const formControlDefaults = {
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
    <Container className="express-web-connect">
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
            const newAccount = accountFromForm(driver)
            API.post('/v1/addAccount', newAccount)
              .then(() => {
                setRedirect(<Redirect to="/settings/accounts" />)
                if (created) {
                  created(newAccount)
                }
              })
              .catch(err => {
                setFeedback((err.response && err.response.data && err.response.data.Error) || "An internal server error occurred")
                if (!err.response || !err.response.data || !err.response.data.Error) {
                  throw err
                }
              })
          }
          setValidated(true)
        }}
      >
        <Row>
          <Form.Label column sm={labelWidth}>Custom name</Form.Label>
          <Col><Form.Control id={makeID("description")} type="text" defaultValue={driver.Description} {...formControlDefaults} required /></Col>
        </Row>
        <Row>
          <Col>
            <RadioGroup
              name={makeID("type")}
              choices={['Bank', 'Credit Card']}
              defaultChoice={isBank ? 'Bank' : 'Credit Card'}
              label="Type of account?"
              smColumns={[labelWidth, inputWidth]}
              onSelect={choice => setIsBank(choice === 'Bank')}
            />
          </Col>
        </Row>
        <Form.Group controlId={makeID("id")} as={Row}>
          <Form.Label column sm={labelWidth}>{isBank ? "Bank Account #" : "Credit Card #"}</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control type="text" {...formControlDefaults} autoFocus required />
          </Col>
        </Form.Group>
        <Form.Group>
          <Form.Group controlId={makeID("username")} as={Row}>
            <Form.Label column sm={labelWidth}>Username</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control
                type="text"
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
                required
                {...formControlDefaults}
              />
              <Form.Control.Feedback type="invalid">
                <p>A password is required when adding a new account</p>
              </Form.Control.Feedback>
            </Col>
          </Form.Group>
        </Form.Group>
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
          <Col><Button type="submit">Add</Button></Col>
        </Form.Row>
      </Form>
    </Container>
  )
}

function formIDFactory() {
  return name => `web-connect-${name}`
}

function accountFromForm(driver) {
  const makeID = formIDFactory()
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
    throw Error("Invalid account type in form: " + formAccountType)
  }
  return {
    AccountID: valueFromID("id"),
    AccountDescription: valueFromID("description"),
    AccountType: accountType,
    WebConnect: {
      Driver: driver.ID.toLowerCase(),
      Data: {
        PasswordConnector: {
          ConnectorUsername: valueFromID("username"),
          ConnectorPassword: valueFromID("password"),
        },
      },
    },
  }
}

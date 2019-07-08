import React from 'react';
import axios from 'axios';
import { Redirect } from 'react-router-dom';
import './Account.css';

import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import Spinner from 'react-bootstrap/Spinner';
import RadioGroup from './RadioGroup';


export default function Account(props) {
  const { account, editable, updated } = props
  const [isBank, setIsBank] = React.useState(null)
  const [validated, setValidated] = React.useState(false)
  const [redirect, setRedirect] = React.useState(null)
  const [verified, setVerified] = React.useState(null)
  const [testFeedback, setTestFeedback] = React.useState(null)
  const [testLoading, setTestLoading] = React.useState(false)

  if (account === null) {
    // prop was defined but hasn't loaded
    return null
  }
  // account prop is either not set or has finished loading
  if (account && isBank === null) {
    setIsBank(account && account.RoutingNumber && account.RoutingNumber !== "")
    return null
  }

  const id = account ? account.ID : 'new'

  const labelWidth = 4
  const inputWidth = 12 - labelWidth
  const makeID = formIDFactory(id)

  const formControlDefaults = {
    disabled: ! editable,
    onBlur: e => {
      e.target.classList.remove('is-valid', 'is-invalid')
      if (e.target.checkValidity() === false) {
        e.target.classList.add('is-invalid')
      } else {
        e.target.classList.add('is-valid')
      }
    },
  }

  const testClicked = () => {
    const form = document.getElementById(makeID("form"))
    if (form.checkValidity() !== false) {
      const newAccount = accountFromForm(id, form)
      setTestLoading(true)
      verifyAccount(newAccount)
        .then(res => {
          setVerified(true)
          setTestFeedback(null)
        })
        .catch(e => {
          // this case should be impossible due to client-side validation
          setVerified(false)
          if (!e.response.data || !e.response.data.Error) {
            throw e
          }
          setTestFeedback(e.response.data.Error)
        })
        .finally(() => setTestLoading(false))
    }
    setValidated(true)
  }

  const testButtonData = {
    props: {
      variant: 'outline-secondary',
      onClick: testClicked,
    },
    text: 'Test',
  }
  if (verified !== null) {
    if (verified) {
      testButtonData.props.variant = 'outline-success'
      testButtonData.text = 'Test Succeeded'
    } else {
      testButtonData.props.variant = 'outline-danger'
      testButtonData.text = 'Test Failed'
    }
  }
  const testButton = (
    <Button {...testButtonData.props}>
      {testButtonData.text}
      {testLoading
        ? <Spinner animation="border" size="sm" className="account-test-spinner" />
        : null
      }
    </Button>
  )

  return (
    <Container className="account">
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
            const newAccount = accountFromForm(id, form)
            updateAccount(account ? account.ID : null, newAccount)
              .then(res => {
                setRedirect(<Redirect to="/accounts" />)
                if (updated) {
                  updated(id, newAccount)
                }
              })
              .catch(e => {
                // this case should be impossible due to client-side validation
                if (e.response.status !== 400) {
                  throw e
                }
                alert(e.response.data.Error)
              })
          }
          setValidated(true)
        }}
        >
        <Form.Group>
          <Row>
            <Col><Form.Control id={makeID("description")} type="text" defaultValue={account ? account.Description : null} {...formControlDefaults} required /></Col>
          </Row>
          <Form.Group controlId={makeID("id")} as={Row}>
            <Form.Label column sm={labelWidth}>Account ID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.ID : null} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <RadioGroup
            choices={['Yes', 'No']}
            defaultChoice={isBank ? 'Yes' : 'No'}
            label="Is this a bank account?"
            onSelect={choice => setIsBank(choice === 'Yes')}
            smColumns={[labelWidth, inputWidth]}
            />
          { !isBank ? null :
            <>
              <Form.Group controlId={makeID("routingNumber")} as={Row}>
                <Form.Label column sm={labelWidth}>Routing number</Form.Label>
                <Col sm={inputWidth}>
                  <Form.Control type="text" defaultValue={account ? account.RoutingNumber : null} {...formControlDefaults} required />
                </Col>
              </Form.Group>
              <RadioGroup
                choices={['Checking', 'Savings']}
                defaultChoice={account ? account.AccountType : null}
                name={makeID("accountType")}
                label="Account type"
                smColumns={[labelWidth, inputWidth]}
                required
                />
            </>
          }
        </Form.Group>

        <Form.Group>
          <Form.Group controlId={makeID("institutionUsername")} as={Row}>
            <Form.Label column sm={labelWidth}>Username</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.Username : null} {...formControlDefaults} required />
              <Form.Control.Feedback type="invalid">
                Please choose a username.
              </Form.Control.Feedback>
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionPassword")} as={Row}>
            <Form.Label column sm={labelWidth}>Password</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control
                type="text"
                placeholder="••••••••"
                required={! account}
                {...formControlDefaults}
                />
              <Form.Control.Feedback type="invalid">
                A password is required when adding a new account
              </Form.Control.Feedback>
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Group>
          <p>To fill out these fields, look up your institution's details on <a href="https://www.ofxhome.com/index.php/home/directory" target="_blank" rel="noopener noreferrer">ofxhome.com</a></p>
          <Form.Group controlId={makeID("institutionDescription")} as={Row}>
            <Form.Label column sm={labelWidth}>Institution name</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.Description : null} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionFID")} as={Row}>
            <Form.Label column sm={labelWidth}>FID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.FID : null} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionOrg")} as={Row}>
            <Form.Label column sm={labelWidth}>Org</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.Org : null} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionURL")} as={Row}>
            <Form.Label column sm={labelWidth}>URL</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="url" defaultValue={account ? account.Institution.URL : null} pattern="https://.*" {...formControlDefaults} required />
              <Form.Control.Feedback type="invalid">
                Provide a valid URL. <code>https://</code> is required.
              </Form.Control.Feedback>
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Group>
          <Form.Group controlId={makeID("institutionClientID")} as={Row}>
            <Form.Label column sm={labelWidth}>Client ID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.ClientID : null} {...formControlDefaults} placeholder="Optional" />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionAppID")} as={Row}>
            <Form.Label column sm={labelWidth}>Client App ID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.AppID : null} placeholder="QWIN" {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionAppVersion")} as={Row}>
            <Form.Label column sm={labelWidth}>Client Version</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.AppVersion : null} placeholder="2500" {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionOFXVersion")} as={Row}>
            <Form.Label column sm={labelWidth}>OFX Version</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.Institution.OFXVersion : null} placeholder="102" {...formControlDefaults} required />
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Row className="account-test">
          <Col sm={labelWidth}>{testButton}</Col>
          { ! testFeedback ? null :
            <Col className="account-test-failed">{testFeedback}</Col>
          }
        </Form.Row>
        &nbsp;
        <Form.Row>
          <Col><Button type="submit">{ account ? 'Save' : 'Add' }</Button></Col>
        </Form.Row>
      </Form>
    </Container>
  )
}

function formIDFactory(accountID) {
  return name => `account-${accountID}-${name}`
}

function accountFromForm(originalAccountID, form) {
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
  return {
    ID: valueFromID("id"),
    Description: valueFromID("description"),
    RoutingNumber: valueFromID("routingNumber"),
    AccountType: valueFromName("accountType"),
    Institution: {
      Description: valueFromID("institutionDescription"),
      FID: valueFromID("institutionFID"),
      Org: valueFromID("institutionOrg"),
      URL: valueFromID("institutionURL"),
      ClientID: valueFromID("institutionClientID"),
      AppID: valueFromID("institutionAppID"),
      AppVersion: valueFromID("institutionAppVersion"),
      OFXVersion: valueFromID("institutionOFXVersion"),
      Username: valueFromID("institutionUsername"),
      Password: valueFromID("institutionPassword"),
    }
  }
}

function updateAccount(originalAccountID, account) {
  if (originalAccountID) {
    return axios.put(`/api/v1/accounts/${originalAccountID}`, account)
  }
  return axios.post(`/api/v1/accounts`, account)
}

function verifyAccount(account) {
  return axios.post(`/api/v1/accounts/${account.ID}/verify`, account)
}

import React from 'react';
import axios from 'axios';
import { Redirect } from 'react-router-dom';
import './Account.css';

import Button from 'react-bootstrap/Button';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Form from 'react-bootstrap/Form';
import RadioGroup from './RadioGroup';


export default function Account(props) {
  const { account, editable, updated } = props
  const [isBank, setIsBank] = React.useState(null)
  const [validated, setValidated] = React.useState(false)
  const [redirect, setRedirect] = React.useState(null)

  if (! account) {
    return null
  }
  if (isBank === null) {
    setIsBank(account && account.RoutingNumber && account.RoutingNumber !== "")
  }

  const labelWidth = 4
  const inputWidth = 12 - labelWidth
  const makeID = formIDFactory(account.ID)

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
  return (
    <Container className="account">
      {redirect}
      <Form
        noValidate
        validated={validated}
        onSubmit={e => {
          e.preventDefault()
          e.stopPropagation()
          const form = e.currentTarget
          if (form.checkValidity() !== false) {
            const newAccount = accountFromForm(account.ID, form)
            updateAccount(account.ID, newAccount)
              .then(res => {
                setRedirect(<Redirect to="/accounts" />)
                if (updated) {
                  updated(account.ID, newAccount)
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
            <Col><Form.Control id={makeID("description")} type="text" defaultValue={account.Description} {...formControlDefaults} required /></Col>
          </Row>
          <Form.Group controlId={makeID("id")} as={Row}>
            <Form.Label column sm={labelWidth}>Account ID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.ID} {...formControlDefaults} required />
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
                  <Form.Control type="text" defaultValue={account.RoutingNumber} {...formControlDefaults} required />
                </Col>
              </Form.Group>
              <RadioGroup
                choices={['Checking', 'Savings']}
                defaultChoice={account.AccountType}
                name={makeID("accountType")}
                label="Account type"
                smColumns={[labelWidth, inputWidth]}
                required
                />
            </>
          }
        </Form.Group>

        <Form.Group>
          <Form.Group controlId={makeID("institutionDescription")} as={Row}>
            <Form.Label column sm={labelWidth}>Institution name</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.Description} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionFID")} as={Row}>
            <Form.Label column sm={labelWidth}>FID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.FID} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionOrg")} as={Row}>
            <Form.Label column sm={labelWidth}>Org</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.Org} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionURL")} as={Row}>
            <Form.Label column sm={labelWidth}>URL</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="url" defaultValue={account.Institution.URL} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionUsername")} as={Row}>
            <Form.Label column sm={labelWidth}>Username</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.Username} {...formControlDefaults} required />
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
                {...formControlDefaults}
                />
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Group>
          <Form.Group controlId={makeID("institutionClientID")} as={Row}>
            <Form.Label column sm={labelWidth}>Client ID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.ClientID} {...formControlDefaults} placeholder="Optional" />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionAppID")} as={Row}>
            <Form.Label column sm={labelWidth}>Client App ID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.AppID} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionAppVersion")} as={Row}>
            <Form.Label column sm={labelWidth}>Client Version</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.AppVersion} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionOFXVersion")} as={Row}>
            <Form.Label column sm={labelWidth}>OFX Version</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account.Institution.OFXVersion} {...formControlDefaults} required />
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Row>
          <Col><Button type="submit">Save</Button></Col>
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
  return axios.put(`/api/v1/accounts/${originalAccountID}`, account)
}

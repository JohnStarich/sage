import axios from 'axios';
import { Link } from "react-router-dom";
import React from 'react';
import Accordion from 'react-bootstrap/Accordion';
import Button from 'react-bootstrap/Button';
import Card from 'react-bootstrap/Card';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import Password from './Password';
import Row from 'react-bootstrap/Row';


const labelWidth = 4
const inputWidth = 12 - labelWidth

export default function FetchAccounts({ created }) {
  const [validated, setValidated] = React.useState(false)
  const [accounts, setAccounts] = React.useState(null)
  const [findFeedback, setFindFeedback] = React.useState(null)
  const [findResult, setFindResult] = React.useState(null)
  const [addFeedback, setAddFeedback] = React.useState(null)
  const [submittingAccounts, setSubmittingAccounts] = React.useState(false)

  if (! created) {
    throw Error("Created prop must be set")
  }

  return (
    <Container>
      <Row><Col><h2>Quick Add</h2></Col></Row>
      <Row>
        <Col>
          <p>Find available accounts for Direct Connect at your institution.</p>
          <p>If you have all of your connection details already, enter them manually <Link to="/accounts/new">here</Link>.</p>
        </Col>
      </Row>
      <Form
        noValidate
        validated={validated}
        onSubmit={e => {
          e.preventDefault()
          e.stopPropagation()
          const form = e.currentTarget
          setValidated(true)
          if (form.checkValidity() !== false) {
            const valueFromID = id => document.getElementById(id).value
            setFindFeedback(null)
            setFindResult(null)
            const password = valueFromID("password")
            axios.post('/api/v1/direct/fetchAccounts', {
              InstDescription: valueFromID("description"),
              InstFID: valueFromID("fid"),
              InstOrg: valueFromID("org"),
              ConnectorURL: valueFromID("url"),
              ConnectorUsername: valueFromID("username"),
              ConnectorPassword: password,
              ConnectorConfig: {
                ClientID: valueFromID("clientID"),
                AppID: valueFromID("appID"),
                AppVersion: valueFromID("appVersion"),
                OFXVersion: valueFromID("ofxVersion"),
              },
            })
              .then(res => {
                res.data.forEach(a => {
                  a.DirectConnect.ConnectorPassword = password // copy in password since API redacts it
                })
                setAccounts(res.data)
                setFindResult("Success! Select desired accounts below:")
              })
              .catch(e => {
                if (!e.response || !e.response.data || !e.response.data.Error) {
                  console.error(e)
                  throw e
                }
                setFindFeedback(e.response.data.Error)
              })
          }
        }}
      >
        <Form.Group controlId="username" as={Row}>
          <Form.Label column sm={labelWidth}>Username</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control type="text" required />
          </Col>
        </Form.Group>
        <Form.Group controlId="password" as={Row}>
          <Form.Label column sm={labelWidth}>Password</Form.Label>
          <Col sm={inputWidth}>
            <Password />
          </Col>
        </Form.Group>

        <Form.Group>
          <Form.Group controlId="description" as={Row}>
            <Form.Label column sm={labelWidth}>Institution name</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" required />
            </Col>
          </Form.Group>

          <p>To look up your institution's FID, Org, and Direct Connect details, visit <a href="https://www.ofxhome.com/index.php/home/directory" target="_blank" rel="noopener noreferrer">ofxhome.com</a>.</p>
          <Form.Group controlId="fid" as={Row}>
            <Form.Label column sm={labelWidth}>FID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" required />
            </Col>
          </Form.Group>

          <Form.Group controlId="org" as={Row}>
            <Form.Label column sm={labelWidth}>Org</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" required />
            </Col>
          </Form.Group>
          <Form.Group controlId="url" as={Row}>
            <Form.Label column sm={labelWidth}>Direct Connect URL</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="url" pattern="(https://|http://localhost).*" required />
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Group>
          <Accordion>
            <Card>
              <Card.Header>
                <Accordion.Toggle as={Button} variant="link" eventKey="0">
                  Advanced Options
                </Accordion.Toggle>
              </Card.Header>
              <Accordion.Collapse eventKey="0">
                <Card.Body>
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
          <Col sm={labelWidth}><Button type="submit">Find</Button></Col>
          <Col>
            <Form.Control.Feedback type="invalid">{findFeedback}</Form.Control.Feedback>
            <p>{findResult}</p>
          </Col>
        </Form.Row>
      </Form>
      &nbsp;

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
              Promise.all(accounts.map(account => {
                const checkbox = document.getElementById("add-" + account.AccountID)
                if (checkbox.checked && !checkbox.disabled) {
                  return axios.post('/api/v1/addAccount', account)
                    .then(res => {
                      checkbox.disabled = true
                      checkbox.classList.add("is-valid")
                      return account
                    })
                }
                return null
              }))
                .then(accounts => {
                  setTimeout(() => {
                    created(...accounts)
                    document.getElementById("return-to-accounts").click()
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
            <Form.Group key={a.AccountID} controlId={"add-" + a.AccountID} as={Row}>
              <Col>
                <Form.Check type="checkbox" label={a.AccountDescription} readOnly={submittingAccounts} />
              </Col>
            </Form.Group>
          )}
          <Form.Row>
            <Col sm={labelWidth}>
              <Button
                variant="outline-secondary"
                disabled={submittingAccounts}
                onClick={() => {
                  accounts.forEach(a => {
                    document.getElementById("add-"+a.AccountID).checked = true
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

import React from 'react';
import './Login.css';

import Alert from 'react-bootstrap/Alert';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import Row from 'react-bootstrap/Row';
import { signIn } from './API';

const labelWidth = 3

function loggedIn() {
  const params = new URLSearchParams(window.location.search)
  const redirect = params.get('redirectURI') || '/web'
  // prevent "back" from returning to the login page
  window.history.replaceState(null, null, redirect)
  window.location.reload()
}

export default function Login() {
  const [validated, setValidated] = React.useState(false)
  const [feedback, setFeedback] = React.useState(null)

  return (
    <Container className="login">
      <Row>
        <Col>
          <h2>Sign In</h2>
        </Col>
      </Row>
        {feedback ? (
          <Row>
            <Col className="login-feedback">
              <Alert variant="danger">
                {feedback.trim().split("\n").map(line =>
                  <span key={line}>{line}<br /></span>
                )}
              </Alert>
            </Col>
          </Row>
        ) : null}
        <Form
          noValidate
          validated={validated}
          onSubmit={e => {
            e.preventDefault()
            e.stopPropagation()
            const form = e.currentTarget
            if (form.checkValidity() !== false) {
              setFeedback(null)
              signIn(form[0].value)
                .then(() => {
                  loggedIn()
                })
                .catch(e => {
                  setFeedback(e.response.data.Error)
                  throw e
                })
            }
            setValidated(true)
          }}
          >
        {/*
        <Form.Group controlId="username" as={Form.Row}>
          <Form.Label column sm={labelWidth}>Username</Form.Label>
          <Col><Form.Control type="text" required /></Col>
        </Form.Group>
        */}
        <Form.Group controlId="password" as={Form.Row}>
          <Form.Label column sm={labelWidth}>Password</Form.Label>
          <Col><Form.Control type="password" required /></Col>
        </Form.Group>
        <Form.Row className="login-submit">
          <Col sm={labelWidth}><Button type="submit">Sign In</Button></Col>
        </Form.Row>
      </Form>
    </Container>
  )
}

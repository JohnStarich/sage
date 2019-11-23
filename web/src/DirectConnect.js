import './DirectConnect.css';
import Accordion from 'react-bootstrap/Accordion';
import Button from 'react-bootstrap/Button';
import Card from 'react-bootstrap/Card';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import LoadingButton from './LoadingButton';
import Password from './Password';
import RadioGroup from './RadioGroup';
import React from 'react';
import Row from 'react-bootstrap/Row';
import API from './API';
import { Redirect } from 'react-router-dom';


export default function(props) {
  const { account, editable, updated } = props
  const [isBank, setIsBank] = React.useState(null)
  const [validated, setValidated] = React.useState(false)
  const [redirect, setRedirect] = React.useState(null)
  const [verified, setVerified] = React.useState(null)
  const [testFeedback, setTestFeedback] = React.useState(null)
  const [institutionURL, setInstitutionURL] = React.useState(null)
  const [directConnectEnabled, setDirectConnectEnabled] = React.useState(null)

  if (account === null) {
    // prop was defined but hasn't loaded
    return null
  }
  // account prop is either not set or has finished loading
  if (account) {
    if (isBank === null) {
      setIsBank(account && (
        (account.RoutingNumber && account.RoutingNumber !== "") ||
        account.AccountType === "assets"
      ))
      return null
    }
    if (directConnectEnabled === null) {
      setDirectConnectEnabled(!!account.DirectConnect)
    }
    if (account.DirectConnect) {
      if (institutionURL === null) {
        setInstitutionURL(account.DirectConnect.ConnectorURL)
        return null
      }
    }
  } else {
    // account is not set, this is for new accounts
    if (directConnectEnabled === null) {
      setDirectConnectEnabled(true)
    }
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

  const processErr = err => {
      setTestFeedback((err.response.data && err.response.data.Error) || "An internal server error occurred")
      if (!err.response.data || !err.response.data.Error) {
        throw err
      }
  }

  const testClicked = () => {
    const form = document.getElementById(makeID("form"))
    if (form.checkValidity() === false) {
      setValidated(true)
      return
    }
    const newAccount = accountFromForm(id, { directConnectEnabled })
    setValidated(true)
    return verifyAccount(newAccount)
      .then(res => {
        setVerified(true)
        setTestFeedback(null)
      })
      .catch(e => {
        // this case should be impossible due to client-side validation
        setVerified(false)
        processErr(e)
      })
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
  const testButton = <LoadingButton {...testButtonData.props}>{testButtonData.text}</LoadingButton>

  let fid = null, org = null, instDescription = null
  if (account) {
    if (account.BasicInstitution) {
      fid = account.BasicInstitution.InstFID
      org = account.BasicInstitution.InstOrg
      instDescription = account.BasicInstitution.InstDescription
    }
    if (account.DirectConnect) {
      fid = account.DirectConnect.InstFID
      org = account.DirectConnect.InstOrg
      instDescription = account.DirectConnect.InstDescription
    }
  }

  return (
    <Container className="direct-connect">
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
            const newAccount = accountFromForm(id, { directConnectEnabled })
            updateAccount(account ? account.AccountID : null, newAccount)
              .then(() => {
                setRedirect(<Redirect to="/accounts" />)
                if (updated) {
                  updated(id, newAccount)
                }
              })
              .catch(e => processErr(e))
          }
          setValidated(true)
        }}
      >
        <Form.Group>
          <Row>
            <Form.Label column sm={labelWidth}>Account Description</Form.Label>
            <Col><Form.Control id={makeID("description")} type="text" defaultValue={account ? account.AccountDescription : null} {...formControlDefaults} required /></Col>
          </Row>
          <RadioGroup
            name={makeID("accountType")}
            choices={['Bank', 'Credit Card']}
            defaultChoice={isBank ? 'Bank' : 'Credit Card'}
            label="Type of account?"
            onSelect={choice => setIsBank(choice === 'Bank')}
            smColumns={[labelWidth, inputWidth]}
          />
          <Form.Group controlId={makeID("id")} as={Row}>
            <Form.Label column sm={labelWidth}>{isBank ? 'Bank account' : 'Credit card'} number</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={account ? account.AccountID : null} {...formControlDefaults} required />
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Group>
          <Form.Group controlId={makeID("institutionDescription")} as={Row}>
            <Form.Label column sm={labelWidth}>Institution name</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={instDescription} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <p>To look up your institution's FID, Org, and Direct Connect details, visit <a href="https://www.ofxhome.com/index.php/home/directory" target="_blank" rel="noopener noreferrer">ofxhome.com</a>.</p>
          <Form.Group controlId={makeID("institutionFID")} as={Row}>
            <Form.Label column sm={labelWidth}>FID</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={fid} {...formControlDefaults} required />
            </Col>
          </Form.Group>

          <Form.Group controlId={makeID("institutionOrg")} as={Row}>
            <Form.Label column sm={labelWidth}>Org</Form.Label>
            <Col sm={inputWidth}>
              <Form.Control type="text" defaultValue={org} {...formControlDefaults} required />
            </Col>
          </Form.Group>
        </Form.Group>

        <Form.Group>
          <RadioGroup
            choices={['Yes', 'No']}
            defaultChoice={directConnectEnabled ? 'Yes' : 'No'}
            label="Use Direct Connect?"
            onSelect={choice => setDirectConnectEnabled(choice === 'Yes')}
            smColumns={[labelWidth, inputWidth]}
          />
          {!directConnectEnabled ? null :
            <Form.Group>
              {!isBank ? null :
                <>
                  <Form.Group controlId={makeID("routingNumber")} as={Row}>
                    <Form.Label column sm={labelWidth}>Routing number</Form.Label>
                    <Col sm={inputWidth}>
                      <Form.Control type="text" defaultValue={account ? account.RoutingNumber : null} {...formControlDefaults} required />
                    </Col>
                  </Form.Group>
                  <RadioGroup
                    choices={['Checking', 'Savings']}
                    defaultChoice={account ? account.BankAccountType : null}
                    name={makeID("bankAccountType")}
                    label="Account type"
                    smColumns={[labelWidth, inputWidth]}
                    required
                  />
                </>
              }

              <Form.Group controlId={makeID("institutionUsername")} as={Row}>
                <Form.Label column sm={labelWidth}>Username</Form.Label>
                <Col sm={inputWidth}>
                  <Form.Control type="text" defaultValue={account && account.DirectConnect ? account.DirectConnect.ConnectorUsername : null} {...formControlDefaults} required />
                  <Form.Control.Feedback type="invalid">
                    Please choose a username.
                  </Form.Control.Feedback>
                </Col>
              </Form.Group>

              <Form.Group controlId={makeID("institutionPassword")} as={Row}>
                <Form.Label column sm={labelWidth}>Password</Form.Label>
                <Col sm={inputWidth}>
                  <Password
                    required={!account && !(institutionURL && institutionURL.startsWith("http://"))}
                    {...formControlDefaults}
                  />
                  <p><em>If your normal password doesn't work, try a PIN instead.</em></p>
                  <Form.Control.Feedback type="invalid">
                    <p>A password is required when adding a new account</p>
                  </Form.Control.Feedback>
                </Col>
              </Form.Group>

              <Form.Group controlId={makeID("institutionURL")} as={Row}>
                <Form.Label column sm={labelWidth}>Direct Connect URL</Form.Label>
                <Col sm={inputWidth}>
                  <Form.Control type="url" defaultValue={institutionURL} pattern="(https://|http://localhost).*" {...formControlDefaults} onChange={e => setInstitutionURL(e.target.value)} required />
                  <Form.Control.Feedback type="invalid">
                    Provide a valid URL. <code>https://</code> is required.
              </Form.Control.Feedback>
                </Col>
              </Form.Group>

              <Form.Group>
                <Accordion>
                  <Card>
                    <Card.Header>
                      <Accordion.Toggle as={Button} variant="link" eventKey="0">
                        More Client Options
                      </Accordion.Toggle>
                    </Card.Header>
                    <Accordion.Collapse eventKey="0">
                      <Card.Body>
                        <Form.Group controlId={makeID("institutionClientID")} as={Row}>
                          <Form.Label column sm={labelWidth}>Client ID</Form.Label>
                          <Col sm={inputWidth}>
                            <Form.Control type="text" defaultValue={account && account.DirectConnect ? account.DirectConnect.ConnectorConfig.ClientID : null} {...formControlDefaults} placeholder="Optional" />
                          </Col>
                        </Form.Group>

                        <Form.Group controlId={makeID("institutionAppID")} as={Row}>
                          <Form.Label column sm={labelWidth}>Client App ID</Form.Label>
                          <Col sm={inputWidth}>
                            <Form.Control type="text" defaultValue={account && account.DirectConnect ? account.DirectConnect.ConnectorConfig.AppID : "QWIN"} placeholder="QWIN" {...formControlDefaults} required />
                          </Col>
                        </Form.Group>

                        <Form.Group controlId={makeID("institutionAppVersion")} as={Row}>
                          <Form.Label column sm={labelWidth}>Client Version</Form.Label>
                          <Col sm={inputWidth}>
                            <Form.Control type="text" defaultValue={account && account.DirectConnect ? account.DirectConnect.ConnectorConfig.AppVersion : "2500"} placeholder="2500" {...formControlDefaults} required />
                          </Col>
                        </Form.Group>

                        <Form.Group controlId={makeID("institutionOFXVersion")} as={Row}>
                          <Form.Label column sm={labelWidth}>OFX Version</Form.Label>
                          <Col sm={inputWidth}>
                            <Form.Control type="text" defaultValue={account && account.DirectConnect ? account.DirectConnect.ConnectorConfig.OFXVersion : "102"} placeholder="102" {...formControlDefaults} required />
                          </Col>
                        </Form.Group>
                      </Card.Body>
                    </Accordion.Collapse>
                  </Card>
                </Accordion>
              </Form.Group>
            </Form.Group>
          }
        </Form.Group>

        {!directConnectEnabled ? null :
          <>
            <Form.Row className="direct-connect-test">
              <Col sm={labelWidth}>{testButton}</Col>
              {!testFeedback ? null :
                <Col className="direct-connect-test-failed">
                  {testFeedback.trim().split("\n").map(line =>
                    <span key={line}>{line}<br /></span>
                  )}
                </Col>
              }
            </Form.Row>
            &nbsp;
          </>
        }
        <Form.Row>
          <Col><Button type="submit">{account ? 'Save' : 'Add'}</Button></Col>
        </Form.Row>
      </Form>
    </Container>
  )
}

function formIDFactory(accountID) {
  return name => `direct-connect-${accountID}-${name}`
}

function accountFromForm(originalAccountID, { directConnectEnabled }) {
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
  let account = {
    AccountID: valueFromID("id"),
    AccountDescription: valueFromID("description"),
  }
  if (directConnectEnabled) {
    account.RoutingNumber = valueFromID("routingNumber")
    account.BankAccountType = valueFromName("bankAccountType")
    account.DirectConnect = {
      InstDescription: valueFromID("institutionDescription"),
      InstFID: valueFromID("institutionFID"),
      InstOrg: valueFromID("institutionOrg"),
      ConnectorURL: valueFromID("institutionURL"),
      ConnectorUsername: valueFromID("institutionUsername"),
      ConnectorPassword: valueFromID("institutionPassword"),
      ConnectorConfig: {
        ClientID: valueFromID("institutionClientID"),
        AppID: valueFromID("institutionAppID"),
        AppVersion: valueFromID("institutionAppVersion"),
        OFXVersion: valueFromID("institutionOFXVersion"),
      },
    }
  } else {
    account.AccountType = valueFromName("accountType") === "Bank" ? "assets" : "liabilities"
    account.BasicInstitution = {
      InstDescription: valueFromID("institutionDescription"),
      InstFID: valueFromID("institutionFID"),
      InstOrg: valueFromID("institutionOrg"),
    }
  }
  return account
}

function updateAccount(originalAccountID, account) {
  if (originalAccountID) {
    return API.post('/v1/updateAccount', Object.assign({}, { AccountID: originalAccountID }, account))
  }
  return API.post('/v1/addAccount', account)
}

function verifyAccount(account) {
  return API.post('/v1/direct/verifyAccount', account)
}

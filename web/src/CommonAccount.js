import React from 'react';
import './CommonAccount.css';

import API from './API';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import ExpressDirectConnect from './ExpressDirectConnect';
import ExpressWebConnect from './ExpressWebConnect';
import Form from 'react-bootstrap/Form';
import Row from 'react-bootstrap/Row';


export default function CommonAccount({ created }) {
  const [name, setName] = React.useState("")
  const [driver, setDriver] = React.useState(null)
  const [suggestions, setSuggestions] = React.useState([])

  React.useEffect(() => {
    if (!name) {
      setSuggestions([])
      return
    }
    Promise.all([
      API.get('/v1/web/getDriverNames', { params: { search: name } })
        .then(res => res.data.DriverNames),
      API.get('/v1/direct/getDrivers', { params: { search: name } })
        .then(res => res.data.Drivers),
    ]).then(([webDrivers, directDrivers]) => {
      setSuggestions(
        directDrivers.concat(
          webDrivers.map(d => ({ ID: d, Description: d, DisplayName: `${d} (beta)` }))
        )
      )
    })
  }, [name])

  return (
    <Container className="common-account">
      <Form.Group as={Row}>
        <Form.Label column sm="4">Search institutions</Form.Label>
        <Col>
          <Form.Control
            autoFocus
            autoCorrect="off"
            spellCheck="false"
            className="institution-name"
            sm="8"
            value={name}
            onChange={e => setName(e.target.value)}
            onKeyDown={e => {
              if (e.keyCode !== 13 || !suggestions || suggestions.length === 0) {
                return // only allow enter when suggestions exist
              }
              e.preventDefault()
              e.target.blur()
              const d = suggestions[0]
              setName(d.Description)
              setDriver(d)
            }}
            />
          <ul className="suggestions">
            {suggestions.map(d =>
              <li key={d.ID} onClick={() => {
                setName(d.Description)
                setDriver(d)
              }}>{d.DisplayName || d.Description}</li>
            )}
          </ul>
        </Col>
      </Form.Group>
      {! driver ? null :
        <div key={driver.ID}>
          <h4 className="institution-title">
            {driver.Description + " - "}
            {driver.ID.startsWith("ofxhome:")
              ? "Direct Connect"
              : <>Web Connect<sup>(beta)</sup></>
            }
          </h4>
          {driver.ID.startsWith("ofxhome:")
            ? <ExpressDirectConnect created={created} driver={driver} />
            : <ExpressWebConnect created={created} driver={driver} />
          }
        </div>
      }
    </Container>
  )
}

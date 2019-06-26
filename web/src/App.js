import React from 'react';
import './App.css';
import 'bootstrap/dist/css/bootstrap.min.css';

import Container from 'react-bootstrap/Container';
import Col from 'react-bootstrap/Col';
import Row from 'react-bootstrap/Row';
import Dropdown from 'react-bootstrap/Dropdown';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';
import Balances from './Balances';
import Expenses from './Expenses';
import Transactions from './Transactions';
import Sync from './Sync';

function App() {
  const [syncTime, setSyncTime] = React.useState(new Date())
  return (
    <div className="App">
      <Navbar bg="dark" expand="sm" variant="dark" sticky="top">
        <Navbar.Brand href="#">Sage</Navbar.Brand>
        <Navbar.Toggle />
        <Navbar.Collapse>
          <Nav className="mr-auto">
            <Nav.Link href="#">Activity</Nav.Link>
          </Nav>
          <Sync className="mr-2" onSync={() => setSyncTime(new Date())} />
          <Dropdown bg="dark" alignRight>
            <Dropdown.Toggle>
              Account
            </Dropdown.Toggle>
            <Dropdown.Menu>
              <Dropdown.Item href="#action/3.1">Account</Dropdown.Item>
              <Dropdown.Item href="#action/3.2">Categories</Dropdown.Item>
            </Dropdown.Menu>
          </Dropdown>
        </Navbar.Collapse>
      </Navbar>
      <Container>
        <Row>
          <Col lg xl={5}><Balances syncTime={syncTime} /></Col>
          <Col xl={7}><Expenses syncTime={syncTime} /></Col>
        </Row>
        <Row>
          <Col><Transactions syncTime={syncTime} /></Col>
        </Row>
      </Container>
    </div>
  );
}

export default App;

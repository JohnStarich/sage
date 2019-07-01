import React from 'react';
import { BrowserRouter as Router, Route, Link, NavLink } from "react-router-dom";
import 'bootstrap/dist/css/bootstrap.min.css';
import './App.css';

import Dropdown from 'react-bootstrap/Dropdown';
import Navbar from 'react-bootstrap/Navbar';
import Nav from 'react-bootstrap/Nav';
import Sync from './Sync';
import Activity from './Activity';
import Accounts from './Accounts';

function App() {
  const [syncTime, setSyncTime] = React.useState(new Date())
  return (
    <Router basename="/web">
      <Navbar className="main-nav" bg="dark" expand="sm" variant="dark" sticky="top">
        <NavLink exact to="/"><Navbar.Brand>Sage</Navbar.Brand></NavLink>
        <Navbar.Toggle />
        <Navbar.Collapse>
          <Nav className="mr-auto">
            <NavLink exact to="/" className="nav-link">Activity</NavLink>
          </Nav>
          <Sync className="mr-2" onSync={() => setSyncTime(new Date())} />
          <Dropdown bg="dark" alignRight>
            <Dropdown.Toggle>
              Settings
            </Dropdown.Toggle>
            <Dropdown.Menu>
              <Link to="/accounts"><Dropdown.Item as="button">Accounts</Dropdown.Item></Link>
              <Link to="/categories"><Dropdown.Item as="button">Categories</Dropdown.Item></Link>
            </Dropdown.Menu>
          </Dropdown>
        </Navbar.Collapse>
      </Navbar>
      
      <Route path="/" exact component={() => <Activity syncTime={syncTime} />} />
      <Route path="/accounts" component={Accounts} />
      <Route path="/categories" component={() => null} />
    </Router>
  );
}

export default App;

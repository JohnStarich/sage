import React from 'react';
import { BrowserRouter as Router, Route, Switch, Link, NavLink } from "react-router-dom";
import 'bootstrap/dist/css/bootstrap.min.css';
import './App.css';

import Accounts from './Accounts';
import Activity from './Activity';
import Breadcrumb from 'react-bootstrap/Breadcrumb';
import Container from 'react-bootstrap/Container';
import Dropdown from 'react-bootstrap/Dropdown';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';
import Sync from './Sync';
import { Crumb, Breadcrumbs } from './Breadcrumb';

function App() {
  return (
    <Router basename="/web">
      <Route path="/" component={AppContent} />
    </Router>
  );
}

export default App;

function AppContent({ match }) {
  const [syncTime, setSyncTime] = React.useState(new Date())
  return (
    <>
      <Crumb title="Sage" match={match} />
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

      <Switch>
        <Route path="/" exact component={() => <Activity syncTime={syncTime} />} />
        <Route>
          <Breadcrumbs as={Breadcrumb} skip={1} render={({ title, match }) =>
              <NavLink className="breadcrumb-item" to={match.url} exact>{title}</NavLink>
            }>
            <Container>
              <Route path="/accounts" component={Accounts} />
              <Route path="/categories" component={() => null} />
            </Container>
          </Breadcrumbs>
        </Route>
      </Switch>
    </>
  )
}

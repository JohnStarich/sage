import React from 'react';
import API from './API';
import { BrowserRouter as Router, Route, Switch, Link, NavLink } from "react-router-dom";
import 'bootstrap/dist/css/bootstrap.min.css';
import './common.css';
import './App.css';

import Accounts from './Accounts';
import Activity from './Activity';
import AdvancedOptions from './AdvancedOptions';
import BalanceSettings from './BalanceSettings';
import Breadcrumb from 'react-bootstrap/Breadcrumb';
import Budgets from './Budgets';
import Categories from './Categories';
import Dropdown from 'react-bootstrap/Dropdown';
import Login from './Login';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';
import Sync from './Sync';
import { Crumb, Breadcrumbs } from './Breadcrumb';
import { ReactComponent as Logo } from './logo/sage.svg';

function App() {
  return (
    <Router basename="/web">
      <Route path="/" component={AppContent} />
    </Router>
  );
}

export default App;

function AppContent({ match }) {
  const [version, setVersion] = React.useState(null)
  const [needsUpdate, setNeedsUpdate] = React.useState(false)
  const [syncTime, setSyncTime] = React.useState(new Date())
  const [appClasses, setAppClasses] = React.useState([])

  React.useEffect(() => {
    API.get('/v1/getVersion')
      .then(res => {
        let version = res.data.Version
        if (version === "") {
          version = "dev"
        }
        setVersion(version)
        setNeedsUpdate(res.data.UpdateAvailable)
      })
    if (navigator.userAgent.includes('Sage/1.0.0') && navigator.userAgent.includes('Macintosh')) {
      setAppClasses(["electron"])
    }
  }, [])

  return (
    <div className={["app"].concat(appClasses).join(" ")}>
      <Crumb title="Sage" match={match} />
      <Navbar className="sage-nav" bg="dark" expand="sm" variant="dark" sticky="top">
        <NavLink exact to="/">
          <Navbar.Brand>
            <Logo className="sage-logo dark" />
            <div className="sage-title">
              <span className="sage-name">Sage</span>
              {needsUpdate
                ? <span className="sage-version needs-update" onClick={() => {
                    window.open("https://github.com/JohnStarich/sage/releases/latest", "_blank")
                  }}>{version}</span>
                : <span className="sage-version">{version}</span>
              }
            </div>
          </Navbar.Brand>
        </NavLink>
        <Navbar.Toggle />
        <Navbar.Collapse>
          <Nav className="mr-auto">
            <NavLink exact to="/" className="nav-link">Activity</NavLink>
            <NavLink exact to="/budgets" className="nav-link">Budgets</NavLink>
          </Nav>
          <Sync className="mr-2" onSync={() => setSyncTime(new Date())} />
          <Dropdown bg="dark" alignRight>
            <Dropdown.Toggle>
              Settings
            </Dropdown.Toggle>
            <Dropdown.Menu>
              <Link to="/accounts"><Dropdown.Item as="button">Accounts</Dropdown.Item></Link>
              <Link to="/balances"><Dropdown.Item as="button">Balances</Dropdown.Item></Link>
              <Link to="/categories"><Dropdown.Item as="button">Categories</Dropdown.Item></Link>
              <Link to="/advanced" className="advanced-settings"><Dropdown.Item as="button">Advanced</Dropdown.Item></Link>
            </Dropdown.Menu>
          </Dropdown>
        </Navbar.Collapse>
      </Navbar>

      <div className="content">
        <Switch>
          <Route path="/" exact component={() => <Activity syncTime={syncTime} />} />
          <Route path="/login" component={Login} />
          <Route path="/budgets" component={Budgets} />
          <Route>
            <Breadcrumbs as={Breadcrumb} skip={1} render={({ title, match }) =>
              <NavLink className="breadcrumb-item" to={match.url} exact>{title}</NavLink>
            }>
              <Route path="/accounts" component={Accounts} />
              <Route path="/balances" component={BalanceSettings} />
              <Route path="/categories" component={Categories} />
              <Route path="/advanced" component={AdvancedOptions} />
            </Breadcrumbs>
          </Route>
        </Switch>
      </div>
    </div>
  )
}

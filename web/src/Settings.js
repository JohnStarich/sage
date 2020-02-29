import React from 'react';
import './Settings.css';

import Accounts from './Accounts';
import AdvancedOptions from './AdvancedOptions';
import BalanceSettings from './BalanceSettings';
import Breadcrumb from 'react-bootstrap/Breadcrumb';
import Categories from './Categories';
import Help from './Help';
import { Breadcrumbs, Crumb } from './Breadcrumb';
import { Link, NavLink, Route, Switch } from "react-router-dom";


export default function ({ match }) {
  return (
    <Breadcrumbs as={Breadcrumb} skip={1} render={({ title, match }) =>
      <NavLink className="breadcrumb-item" to={match.url} exact>{title}</NavLink>
    }>
      <Crumb title="Settings" match={match} />
      <Switch>
        <Route path={`${match.path}/accounts`} component={Accounts} />
        <Route path={`${match.path}/balances`} component={BalanceSettings} />
        <Route path={`${match.path}/categories`} component={Categories} />
        <Route path={`${match.path}/advanced`} component={AdvancedOptions} />
        <Route path={`${match.path}/help`} component={Help} />
        <Route exact path={match.path}>
          <ul className="settings-tiles">
            <li>
              <Link to={`${match.url}/accounts`}>
                <h2>Accounts</h2>
                <p>Add and update bank or credit card accounts.</p>
              </Link>
            </li>
            <li>
              <Link to={`${match.url}/balances`}>
                <h2>Balances</h2>
                <p>Edit starting balances for accounts.</p>
              </Link>
            </li>
            <li>
              <Link to={`${match.url}/categories`}>
                <h2>Categories</h2>
                <p>Add and edit transaction categories.</p>
              </Link>
            </li>
            <li>
              <Link to={`${match.url}/advanced`} className="advanced-settings">
                <h2>Advanced</h2>
                <p>Power user tools.</p>
              </Link>
            </li>
            <li>
              <Link to={`${match.url}/help`}>
                <h2>Help &amp; Feedback</h2>
                <p>Send suggestions, request features, or ask questions.</p>
              </Link>
            </li>
          </ul>
        </Route>
      </Switch>
    </Breadcrumbs>
  )
}

import './Activity.css';
import React from 'react';

import Container from 'react-bootstrap/Container';
import Col from 'react-bootstrap/Col';
import Row from 'react-bootstrap/Row';

import Balances from './Balances';
import Expenses from './Expenses';
import Transactions from './Transactions';

export default function Activity(props) {
  const { syncTime } = props;
  return (
    <Container className="content">
      <Row>
        <Col lg xl={5}><Balances syncTime={syncTime} /></Col>
        <Col xl={7}><Expenses syncTime={syncTime} /></Col>
      </Row>
      <Row>
        <Col><Transactions syncTime={syncTime} /></Col>
      </Row>
    </Container>
  )
}

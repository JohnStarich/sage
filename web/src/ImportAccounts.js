import API from './API';
import React from 'react';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import Row from 'react-bootstrap/Row';

export default function ImportAccounts() {
  return (
    <Container>
      <Row><Col><h2>Import</h2></Col></Row>
      <Row>
        <Col>
          <p>Import OFX or QFX files. Typically, you can download these from your financial institution's "Quicken" or "Microsoft Money" downloads.</p>
        </Col>
      </Row>
      <Form
        noValidate
        onSubmit={e => {
          e.preventDefault()
          e.stopPropagation()
          const form = e.currentTarget
          if (form.checkValidity() !== false) {
            const formFiles = form.querySelector('[type=file]')
            const files = formFiles.files;
            if (files.length !== 1) {
              throw Error("Must provide one file to import")
            }
            API.post('/v1/importOFX', files[0])
              .then(() => window.location.reload())
              .catch(e => {
                if (!e.response.data || !e.response.data.Error) {
                  throw e
                }
                alert(e.response.data.Error)
              })
          }
        }}
      >
        <Form.Row>
          <Form.Control type="file" required />
        </Form.Row>
        &nbsp;
        <Form.Row>
          <Col><Button type="submit">Import</Button></Col>
        </Form.Row>
      </Form>
    </Container>
  )
}  

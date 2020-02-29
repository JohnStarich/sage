import React from 'react';
import './RadioGroup.css';

import Form from 'react-bootstrap/Form';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';


export default function RadioGroup(props) {
  const {
    choices,
    defaultChoice,
    disabledChoices = [],
    label,
    name,
    onSelect,
    smColumns,
    ...remainingProps
  } = props;
  if (!choices || ! Array.isArray(choices)) {
    throw new Error("Choices array must be provided")
  }
  if (smColumns && smColumns.length !== 2) {
    throw new Error("smColumns prop must be an array of 2 column widths")
  }

  const [id] = React.useState(name || `radio-group-${Math.random().toString()}`)

  const ColTag = smColumns ? Col : 'div'
  const disabledSet = new Set(disabledChoices)

  return (
    <Form.Group className="radio-group" as={smColumns ? Row : undefined}>
      {!label ? null :
        <Form.Label htmlFor={id} column={smColumns} sm={smColumns ? smColumns[0] : null}>
          {label}
        </Form.Label>
      }
      <ColTag sm={smColumns ? smColumns[1] : null}>
        {choices.map((choice, i) =>
          <Form.Check key={choice} inline id={`${id}-${choice}`}>
            <Form.Check.Input
              defaultChecked={defaultChoice && defaultChoice.toUpperCase() === choice.toUpperCase()}
              disabled={disabledSet.has(choice)}
              name={id}
              onChange={e => onSelect && onSelect(e.target.value)}
              type="radio"
              value={choice}
              {...remainingProps}
            />
            <Form.Check.Label className="btn btn-outline-secondary">{choice}</Form.Check.Label>
            {i !== choices.length - 1 ? null :
              <Form.Control.Feedback type="invalid">Required</Form.Control.Feedback>
            }
          </Form.Check>
        )}
      </ColTag>
    </Form.Group>
  )
}

import React from 'react';
import './Amount.css';
import Form from 'react-bootstrap/Form';

export default function(props) {
  const TagName = props.tagName || 'span';
  let amount = props.amount
  if (typeof amount !== 'number') {
    return "NaN"
  }
  let className = "amount"
  if (amount < 0) {
    className += " amount-negative"
  } else {
    className += " amount-positive"
  }
  if (props.className) {
    className += " " + props.className
  }

  if (props.editable) {
    if (! props.onChange) {
      throw Error("Editable amounts must have an onChange prop")
    }
    const [amount, setAmount] = React.useState(props.amount)
    const onAmountChange = e => {
      let amountStr = e.target.value
      if (props.prefix && props.prefix.length > 0 && amountStr.startsWith(props.prefix)) {
        // trim off prefix
        amountStr = amountStr.slice(props.prefix.length)
      }

      let amountNum = Number(amountStr)
      if (amountStr === '-') {
        // if input is just negative, assume it's 0
        amountNum = 0
      }
      if (Number.isNaN(amountNum)) {
        return
      }
      // TODO limit to two decimal places
      if (amountNum === amount) {
        setAmount(amountStr)
        return
      }
      props.onChange(amountNum)
      setAmount(amountStr)
    }
    return (
      <Form.Control
        className={className}
        type="text"
        disabled={props.disabled}
        value={`${props.prefix}${amount}`}
        onChange={onAmountChange}
        />
    )
  }

  let sign = ""
  let [integer, fractional] = amount.toFixed(2).split(".")
  if (amount < 0) {
    sign = "-"
    integer = integer.slice(1)
  }

  let newAmount = Array.from(integer)
    .reverse()
    .map((ch, i) => {
      if (i !== 0 && i !== integer.length && i % 3 === 0) {
        return ch + "," 
      }
      return ch
    })
    .reverse()
    .join("")
  let commaBlocks = newAmount.split(',')
  return (
    <TagName className={className}>
      <TagName className="amount-prefix">{props.prefix}</TagName>
      <TagName className="amount-sign">{sign}</TagName>
      {commaBlocks[0]}
      {commaBlocks.slice(1).map((group, i) =>
        <TagName key={i}>
          <TagName className="amount-thousands">,</TagName>
          {group}
        </TagName>
      )}
      {"."+fractional}
    </TagName>
  )
}

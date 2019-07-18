import React from 'react';
import './Amount.css';
import Form from 'react-bootstrap/Form';

export default function(props) {
  const {
    tagName,
    amount,
    className,
    editable,
    onChange,
    prefix,
    ...remainingProps
  } = props
  const TagName = tagName || 'span';
  if (typeof amount !== 'number') {
    return "NaN"
  }

  let fullClassName = "amount monospace"
  if (amount < 0) {
    fullClassName += " amount-negative"
  } else {
    fullClassName += " amount-positive"
  }
  if (className) {
    fullClassName += " " + className
  }


  if (editable) {
    if (! onChange) {
      throw Error("Editable amounts must have an onChange prop")
    }
    const [currentAmount, setCurrentAmount] = React.useState(amount)
    const onAmountChange = e => {
      let amountStr = e.target.value
      if (prefix && prefix.length > 0 && amountStr.startsWith(prefix)) {
        // trim off prefix
        amountStr = amountStr.slice(prefix.length)
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
      if (amountNum === currentAmount) {
        setCurrentAmount(amountStr)
        return
      }
      onChange(amountNum)
      setCurrentAmount(amountStr)
    }
    const propSet = new Set(['value', 'defaultValue'])
    Object.keys(remainingProps).forEach(p => {
      if (propSet.has(p)) {
        throw new Error("Invalid prop for Amount: " + p)
      }
    })

    return (
      <Form.Control
        className={fullClassName}
        type="text"
        value={`${prefix}${currentAmount}`}
        onChange={onAmountChange}
        {...remainingProps}
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
    <TagName className={fullClassName}>
      <TagName className="amount-prefix">{prefix}</TagName>
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

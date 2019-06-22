import React from 'react';
import './Amount.css';

export default function(props) {
  const TagName = props.tagName || 'span';
  let amount = props.amount
  if (typeof amount !== 'number') {
    return "NaN"
  }
  let className = "amount"
  let sign = ""
  let [integer, fractional] = amount.toFixed(2).split(".")
  if (amount < 0) {
    sign = "-"
    integer = integer.slice(1)
    if (props.highlightNegative) {
      className += " amount-negative"
    }
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

import React from 'react';
import Form from 'react-bootstrap/Form';

function redact(s) {
  return '•'.repeat(s.length)
}

// a password input field that doesn't trigger browser password saving
export default function Password({ defaultValue = "", onChange, id, name, ...remainingProps }) {
  const [value, setValue] = React.useState(defaultValue)
  const [fakeValue, setFakeValue] = React.useState(redact(defaultValue))
  const inputChanged = e => {
    const inputValue = e.target.value
    let newValue = value
    if (value.length < inputValue.length) {
      let appendValue = inputValue.slice(value.length)
      newValue = newValue + appendValue
    } else if (value.length > inputValue.length) {
      newValue = newValue.slice(0, inputValue.length)
    }
    setValue(newValue)
    setFakeValue(redact(newValue))
    if (onChange) {
      onChange(newValue)
    }
  }
  const selectStart = e => {
    const value = e.target.value
    if (e.target.selectionStart !== 0 || e.target.selectionEnd !== value.length) {
      e.target.setSelectionRange(e.target.value.length, e.target.value.length)
      e.preventDefault()
      return false
    }
  }
  const preventDefault = e => {
    e.preventDefault()
    return false
  }
  return (
    <>
      <Form.Control
        id={id}
        name={name}
        style={{display: "none"}}
        value={value}
        readOnly
      />
      <Form.Control
        {...remainingProps}
        autoComplete="off"
        type="text"
        value={fakeValue}
        onChange={inputChanged}
        onSelect={selectStart}
        onCut={preventDefault}
        onCopy={preventDefault}
      />
    </>
  )
}

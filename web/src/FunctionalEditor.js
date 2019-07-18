import React from 'react';


// FunctionalEditor returns a react-bootstrap-table compatible editor component
// Pass the unfurled "props" arg from 'editorRenderer' to this component
// Use 'onChange' prop to call 'onUpdate' when editing is complete. Otherwise,
// use the built-in Enter or onBlur functionality of the editor to save values.
export default class FunctionalEditor extends React.Component {
  editableState = null

  constructor(props) {
    super(props)
    this.editableState = props.defaultValue || null
  }

  getValue() {
    return this.editableState
  }

  render() {
    const {
      children,
      defaultValue: _,
      onChange,
      onUpdate: __,
      value: ___,
      ...remainingProps
    } = this.props

    let child = React.Children.only(children)
    return React.cloneElement(child, {
      onChange: value => {
        this.editableState = value
        if (onChange) {
          onChange(value)
        }
      },
      ...remainingProps
    })
  }
}

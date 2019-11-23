import './CategoryPicker.css';
import React from 'react';
import API from './API';
import Form from 'react-bootstrap/Form';


export function cleanCategory(account) {
  let i = account.lastIndexOf(":")
  if (i === -1) {
    return account
  }
  return account.slice(i + 1)
}

let categoriesPromise = null

export const Categories = () => {
  if (categoriesPromise === null) {
    categoriesPromise = API.get('/v1/getCategories')
      .then(res => res.data.Accounts)
      .then(accounts =>
        accounts.map(c => [c, c.replace(/:/g, ' > ')]))
  }
  return categoriesPromise
}

export function CategoryPicker({ category, setCategory, filter, disabled }) {
  if (!setCategory) {
    throw Error("setCategory is required")
  }
  const [categories, setCategories] = React.useState([])
  React.useEffect(() => {
    Categories().then(allCategories => {
      let newCategories = allCategories
      if (filter) {
        newCategories = allCategories.filter(c => filter(c[0]))
      }
      if (newCategories.length !== 0 && category === null) {
        setCategory(newCategories[0][0])
      }
      setCategories(newCategories)
    })
  }, [category, filter, setCategory])
  if (categories.length === 0) {
    return null
  }
  return (
    <Form.Control
      as="select"
      disabled={disabled}
      value={category || ""}
      onChange={e => setCategory(e.target.value)}
      className="category">
      {categories.map(c =>
        <option key={c[0]} value={c[0]}>{c[1]}</option>
      )}
    </Form.Control>
  )
}

export default CategoryPicker;

import './CategoryPicker.css';
import React from 'react';
import API from './API';
import Dropdown from 'react-bootstrap/Dropdown';


export function cleanCategory(account) {
  let i = account.lastIndexOf(":")
  if (i === -1) {
    return account
  }
  return account.slice(i + 1)
}

let categoriesPromise = null

function render(category) {
  return category.replace(/:/g, ' > ')
}

export const Categories = () => {
  if (categoriesPromise === null) {
    categoriesPromise = API.get('/v1/getCategories')
      .then(res => res.data.Accounts)
      .then(accounts =>
        accounts.map(c => [c, render(c)]))
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
    <Dropdown
      disabled={disabled}
      className="category-picker"
      onSelect={(_, e) => setCategory(e.target.getAttribute('value'))}
      >
      <Dropdown.Toggle variant="secondary" className="category">{render(category)}</Dropdown.Toggle>
      <Dropdown.Menu>
        {categories.map(c =>
          <Dropdown.Item key={c[0]} value={c[0]} className="category">{c[1]}</Dropdown.Item>
        )}
      </Dropdown.Menu>
    </Dropdown>
  )
}

export default CategoryPicker;

import './CategoryPicker.css';
import React from 'react';
import API from './API';
import Dropdown from 'react-bootstrap/Dropdown';
import Form from 'react-bootstrap/Form';


export function cleanCategory(account) {
  return splitCategory(account).base
}

function splitCategory(account) {
  let i = account.lastIndexOf(":")
  if (i === -1) {
    return {
      base: account,
      parent: account,
    }
  }
  return {
    base: account.slice(i + 1),
    parent: account.slice(0, i),
  }
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

function clearCategoryCache() {
  categoriesPromise = null
}

export function CategoryPicker({ id, category, setCategory, filter, disabled }) {
  if (!setCategory) {
    throw Error("setCategory is required")
  }
  const [search, setSearch] = React.useState("")

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

  let displayCategories = categories.map(c => c[0])
  if (search) {
    displayCategories = displayCategories.filter(c => c.includes(search)).sort()
  }

  let newCategory = search ? search.toLocaleLowerCase().replace(/\s+/g, " ") : ""
  if (!newCategory.startsWith("expenses:") && !newCategory.startsWith("revenues:")) {
    newCategory = "expenses:" + newCategory
  }
  return (
    <Dropdown
      disabled={disabled}
      className="category-picker"
      onSelect={(_, e) => setCategory(e.currentTarget.getAttribute('value'))}
      >
      <Dropdown.Toggle variant="outline-secondary" id={id}>
        <Category value={category} />
      </Dropdown.Toggle>
      <Dropdown.Menu>
          <Form.Control
            type="search"
            placeholder="Search..."
            autoFocus
            value={search}
            onChange={e => setSearch(e.target.value)}
            onKeyDown={e => {
              if (e.key !== 'Enter') {
                return
              }
              if (displayCategories.length !== 0) {
                setCategory(displayCategories[0])
              } else if (newCategory !== "") {
                setCategory(newCategory)
                clearCategoryCache()
              }
            }}
            />
        {displayCategories.map(c =>
          <Dropdown.Item key={c} value={c}><Category value={c} /></Dropdown.Item>
        )}
        {search ?
          <div className="new-category">
            <div className="new-category-prompt"><em>Create new:</em></div>
            <Dropdown.Item key={newCategory} value={newCategory}><Category value={newCategory} /></Dropdown.Item>
          </div>
        : null}
      </Dropdown.Menu>
    </Dropdown>
  )
}

export default CategoryPicker;

export function Category({
  value: category,
  className,
}) {
  if (! category) {
    return null
  }
  const { base, parent } = splitCategory(category)
  const trimmedParent = parent.replace(/^(expenses|revenues):/, "")
  const classNames = ["category"]
  if (className) {
    classNames.push(className)
  }
  return (
    <div className={classNames.join(" ")}>
      <div className="category-name">{base}</div>
      <div className="category-id">{trimmedParent}</div>
    </div>
  )
}

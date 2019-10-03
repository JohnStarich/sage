import './Breadcrumb.css';
import React from 'react';
import withSideEffect from 'react-side-effect';

let breadcrumbs = []

const InnerCrumbs = React.createContext(null)

export function Breadcrumbs(props) {
  const { render, as = 'div', skip = 0, ...remainingProps } = props

  const [renderCrumbs, setRenderCrumbs] = React.useState([])
  const [lastUpdate, setLastUpdate] = React.useState(false)
  const breadcrumbsCopy = breadcrumbs // used to fix static dependency analysis
  React.useEffect(() => {
    setRenderCrumbs(breadcrumbs.slice(skip))
  }, [breadcrumbsCopy, lastUpdate, skip])

  if (!render) {
    throw new Error("Must include render prop")
  }

  const updateCrumbs = () => {
    // trigger re-render whenever a crumb's props are updated
    setLastUpdate(!lastUpdate)
  }

  const Tag = as
  return (
    <InnerCrumbs.Provider value={updateCrumbs}>
      <Tag className="breadcrumb-bar" {...remainingProps}>
        {renderCrumbs.map((crumb, i) =>
          React.cloneElement(render(crumb), { key: i })
        )}
      </Tag>

      {props.children}
    </InnerCrumbs.Provider>
  )
}


function reducePropsToState(propsList) {
  let crumbs = propsList
  let title = propsList.reduce((acc, { title, separator = ' | ' }, i) => {
    if (i !== 0) {
      acc += separator
    }
    if (title) {
      acc += title
    }
    return acc
  }, "");
  return { title, crumbs }
}

function handleStateChangeOnClient({ title, crumbs }) {
  document.title = title;
  breadcrumbs = crumbs
}

// Crumb must render for the given route *unconditionally* in order for the breadcrumb component to work correctly.
// withSideEffect reduces updates for crumbs mounted before the Breadcrumbs component
export const Crumb = withSideEffect(
  reducePropsToState,
  handleStateChangeOnClient
)(props => {
  const crumbContext = React.useContext(InnerCrumbs)
  const [lastProps, setLastProps] = React.useState(props)
  if (crumbContext !== null && lastProps !== props) {
    // if this crumb mounts after the breadcrumb component, call the update function
    setLastProps(props)
    crumbContext()
  }
  return null
});

export default Crumb;

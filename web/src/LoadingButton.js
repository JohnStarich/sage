import React from 'react';
import Button from 'react-bootstrap/Button';
import Spinner from 'react-bootstrap/Spinner';
import './LoadingButton.css';


export default function LoadingButton(props) {
  const {
    className,
    onClick,
    children,
    ...remainingProps
  } = props
  const [isLoading, setIsLoading] = React.useState(false)

  let fullClassName = "loading-btn"
  if (className) {
    fullClassName += " " + className
  }

  return (
    <Button
      className={fullClassName}
      onClick={() => {
        setIsLoading(true)
        Promise.resolve(onClick())
          .finally(() => setIsLoading(false))
      }}
      {...remainingProps}
    >
      {children}
      {isLoading
        ? <Spinner animation="border" size="sm" className="loading-btn-spinner" />
        : null
      }
    </Button>
  )
}

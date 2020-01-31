import React from 'react';
import DatePicker from 'react-datepicker';


function convertUTCToLocalDate(date) {
  if (!date) {
    return date
  }
  date = new Date(date)
  date = new Date(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate())
  return date
}

function convertLocalToUTCDate(date) {
  if (!date) {
    return date
  }
  date = new Date(date)
  date = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  return date
}

export default function UTCDatePicker({
  startDate,
  endDate,
  selected,
  onChange,
  ...props
}) {
  return (
    <DatePicker
      startDate={convertUTCToLocalDate(startDate)}
      endDate={convertUTCToLocalDate(endDate)}
      selected={convertUTCToLocalDate(selected)}
      onChange={date => onChange(convertLocalToUTCDate(date))}
      {...props}
    />
  )
}

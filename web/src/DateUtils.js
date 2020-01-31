
export function firstOfMonth(date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), 1))
}

export function lastOfMonth(date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + 1, 0))
}

export function someMonthsAgo(date, months = 12) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() - (months - 1), 1))
}

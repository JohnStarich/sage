import React from 'react';

const initialColors = [
  '#1e90ff',
  '#ff56a1',
  '#95Eb6f',
  '#008b8b',
]
const brightnesses = [
  1,
  0.9,
  0.8,
  0.7,
  0.6,
]

const orderedColors = (() => {
  let allColors = []
  for (let color of initialColors) {
    for (let brightness of brightnesses) {
        allColors.push(scaleColor(color, brightness))
    }
  }
  return allColors
})()

const colors = (() => {
  let result = []
  let rows = initialColors.length
  let cols = brightnesses.length
  let row = 0, col = 0
  for (let i = 0; i < orderedColors.length; i++) {
    // advance one to the right and up, reflecting if necessary
    // uses the "Diagonal switchback pattern" from here: https://medium.com/design-ibm/inclusive-color-sequences-for-data-viz-in-6-steps-712869b910c2
    result.push(orderedColors[col + row * cols])
    col++
    row--
    if (row < 0) {
      col = (cols - col) % cols
      row = (row + rows) % rows
    }
    if (col >= cols) {
      col = col % cols
      row = (rows - row - 1) % rows
    }
  }
  return result
})()

function scaleColor(color, scalar) {
  let value = parseInt(color.slice(1), 16)
  let red = (value >> 16) & 0xFF
  let green = (value >> 8) & 0xFF
  let blue = value & 0xFF
  // scale and cap brightness at maximum without distorting color
  if (scalar > 1) {
    let maxCurrentComponent = Math.max(red, green, blue) / 255
    scalar = 1 + (1 - maxCurrentComponent) / scalar
  }
  red = Math.min(red * scalar, 0xFF) << 16
  green = Math.min(green * scalar, 0xFF) << 8
  blue = Math.min(blue * scalar, 0xFF)
  let newColor = red | green | blue
  return '#' + newColor.toString(16).padStart(6, '0')
}

export default colors;

function makeColorPalette(colors) {
  return (
    <div style={{
      display: 'flex',
      flexWrap: 'wrap',
      width: brightnesses.length * 100,
    }}>
      {colors.map((c, i) =>
        <div key={i} style={{
          width: 100,
          height: 100,
          backgroundColor: c,
        }} />
      )}
    </div>
  )
}

export function ColorPalette() {
  return (
    <div>
      <h2>Ordered Palette</h2>
      {makeColorPalette(orderedColors)}
      <h2>Shuffled Palette</h2>
      {makeColorPalette(colors)}
    </div>
  )
}

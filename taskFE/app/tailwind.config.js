/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.vue",
    "./src/**/*.js",
    "./src/**/*.html",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#7241F2',
        secondary: '#F1F2F6',
        success: '#10b981',
        danger: '#FF324D',
        warning: '#f59e0b',
      },
    },
  },
  plugins: [],
}
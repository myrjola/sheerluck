/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './ui/html/**/*.{html,js,gohtml}',
  ],
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
  ],
}


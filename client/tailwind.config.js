/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          light: '#25D366',
          DEFAULT: '#128C7E',
          dark: '#075E54',
          accent: '#34B7F1'
        }
      }
    },
  },
  plugins: [],
}

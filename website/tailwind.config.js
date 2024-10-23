/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.tsx"],
  corePlugins: {
    preflight: false,
    container: false,
  },
  darkMode: ["class", '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        magenta: {
          DEFAULT: "#E20074",
          hover: "#c00063",
          active: "#9e0051"
        },
      },
    },
    container: {
      center: true,
    },
  },
  plugins: [],
}

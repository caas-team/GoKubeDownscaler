/** @type {import('tailwindcss').Config} */
// eslint-disable-next-line no-undef
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
          active: "#9e0051",
        },
      },
    },
    container: {
      center: true,
    },
  },
  plugins: [],
};

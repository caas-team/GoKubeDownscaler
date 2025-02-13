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
        kblue: {
          DEFAULT: "#326CE5",
          50: "#f0f6fe",
          100: "#dceafd",
          200: "#c1dcfc",
          300: "#97c6f9",
          400: "#65a8f5",
          500: "#4186f0",
          600: "#326ce5",
          700: "#2353d2",
          800: "#2344aa",
          900: "#223d86",
          950: "#192752",
        },
      },
    },
    container: {
      center: true,
    },
  },
  plugins: [],
};

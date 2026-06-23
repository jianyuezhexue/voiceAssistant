/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'bg-deep': '#050508',
        'bg-surface': '#0c0c14',
        'bg-elevated': '#12121f',
        'accent-cyan': '#00f5d4',
        'accent-purple': '#9d4edd',
        'accent-pink': '#ff006e',
      },
      fontFamily: {
        display: ['Space Grotesk', 'system-ui', 'sans-serif'],
        body: ['DM Sans', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}

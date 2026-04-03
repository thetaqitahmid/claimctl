/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}"
  ],
  theme: {
    extend: {
      colors: {
        slate: {
          950: '#020617',
        },
        brand: {
          available: '#10b981', // Emerald 500
          busy: '#f43f5e',      // Rose 500
          queued: '#3b82f6',     // Blue 500
          amber: '#f59e0b',     // Amber 500
        }
      },
      backdropBlur: {
        xs: '2px',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      }
    },
  },
  plugins: [],
};


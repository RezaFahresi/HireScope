/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#1E40AF',
          50: '#EFF6FF',
          100: '#DBEAFE',
          200: '#BFDBFE',
          300: '#93C5FD',
          400: '#60A5FA',
          500: '#3B82F6',
          600: '#2563EB',
          700: '#1D4ED8',
          800: '#1E40AF',
          900: '#1E3A8A',
        },
        secondary: '#3B82F6',
        cta: '#F59E0B',
        brand: {
          bg: '#F8FAFC',
          text: '#1E3A8A',
          border: '#E2E8F0',
        },
      },
      fontFamily: {
        heading: ['"Fira Code"', 'monospace'],
        body: ['"Fira Sans"', 'sans-serif'],
        sans: ['"Fira Sans"', 'sans-serif'],
        mono: ['"Fira Code"', 'monospace'],
      },
      spacing: {
        xs: '4px',
        sm: '8px',
        md: '16px',
        lg: '24px',
        xl: '32px',
        '2xl': '48px',
        '3xl': '64px',
      },
      boxShadow: {
        sm: '0 1px 2px rgba(0,0,0,0.05)',
        md: '0 4px 6px rgba(0,0,0,0.1)',
        lg: '0 10px 15px rgba(0,0,0,0.1)',
        xl: '0 20px 25px rgba(0,0,0,0.15)',
      },
      transitionDuration: {
        fast: '150ms',
        base: '200ms',
        slow: '300ms',
      },
      borderRadius: {
        card: '12px',
        modal: '16px',
      },
      maxWidth: {
        dashboard: '1400px',
      },
    },
  },
  plugins: [],
}

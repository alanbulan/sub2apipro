/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      ringColor: {
        DEFAULT: 'var(--border-color)'
      },
      colors: {
        primary: {
          DEFAULT: 'var(--color-primary)',
          50: 'rgb(var(--primary-50-rgb) / <alpha-value>)',
          100: 'rgb(var(--primary-100-rgb) / <alpha-value>)',
          200: 'rgb(var(--primary-200-rgb) / <alpha-value>)',
          300: 'rgb(var(--primary-300-rgb) / <alpha-value>)',
          400: 'rgb(var(--primary-400-rgb) / <alpha-value>)',
          500: 'rgb(var(--primary-500-rgb) / <alpha-value>)',
          600: 'rgb(var(--primary-600-rgb) / <alpha-value>)',
          700: 'rgb(var(--primary-700-rgb) / <alpha-value>)',
          800: 'rgb(var(--primary-800-rgb) / <alpha-value>)',
          900: 'rgb(var(--primary-900-rgb) / <alpha-value>)',
          950: 'rgb(var(--primary-950-rgb) / <alpha-value>)'
        },
        accent: {
          DEFAULT: 'var(--accent-base)',
          50: 'rgb(var(--accent-50-rgb) / <alpha-value>)',
          100: 'rgb(var(--accent-100-rgb) / <alpha-value>)',
          200: 'rgb(var(--accent-200-rgb) / <alpha-value>)',
          300: 'rgb(var(--accent-300-rgb) / <alpha-value>)',
          400: 'rgb(var(--accent-400-rgb) / <alpha-value>)',
          500: 'rgb(var(--accent-500-rgb) / <alpha-value>)',
          600: 'rgb(var(--accent-600-rgb) / <alpha-value>)',
          700: 'rgb(var(--accent-700-rgb) / <alpha-value>)',
          800: 'rgb(var(--accent-800-rgb) / <alpha-value>)',
          900: 'rgb(var(--accent-900-rgb) / <alpha-value>)',
          950: 'rgb(var(--accent-950-rgb) / <alpha-value>)'
        },
        dark: {
          DEFAULT: 'var(--dark-base)',
          50: 'rgb(var(--dark-50-rgb) / <alpha-value>)',
          100: 'rgb(var(--dark-100-rgb) / <alpha-value>)',
          200: 'rgb(var(--dark-200-rgb) / <alpha-value>)',
          300: 'rgb(var(--dark-300-rgb) / <alpha-value>)',
          400: 'rgb(var(--dark-400-rgb) / <alpha-value>)',
          500: 'rgb(var(--dark-500-rgb) / <alpha-value>)',
          600: 'rgb(var(--dark-600-rgb) / <alpha-value>)',
          700: 'rgb(var(--dark-700-rgb) / <alpha-value>)',
          800: 'rgb(var(--dark-800-rgb) / <alpha-value>)',
          900: 'rgb(var(--dark-900-rgb) / <alpha-value>)',
          950: 'rgb(var(--dark-950-rgb) / <alpha-value>)'
        },
        gray: {
          DEFAULT: 'var(--muted)',
          50: 'rgb(var(--gray-50-rgb) / <alpha-value>)',
          100: 'rgb(var(--gray-100-rgb) / <alpha-value>)',
          200: 'rgb(var(--gray-200-rgb) / <alpha-value>)',
          300: 'rgb(var(--gray-300-rgb) / <alpha-value>)',
          400: 'rgb(var(--gray-400-rgb) / <alpha-value>)',
          500: 'rgb(var(--gray-500-rgb) / <alpha-value>)',
          600: 'rgb(var(--gray-600-rgb) / <alpha-value>)',
          700: 'rgb(var(--gray-700-rgb) / <alpha-value>)',
          800: 'rgb(var(--gray-800-rgb) / <alpha-value>)',
          900: 'rgb(var(--gray-900-rgb) / <alpha-value>)',
          950: 'rgb(var(--gray-950-rgb) / <alpha-value>)'
        },
        slate: { DEFAULT: 'var(--muted)', 50: 'rgb(var(--gray-50-rgb) / <alpha-value>)', 100: 'rgb(var(--gray-100-rgb) / <alpha-value>)', 200: 'rgb(var(--gray-200-rgb) / <alpha-value>)', 300: 'rgb(var(--gray-300-rgb) / <alpha-value>)', 400: 'rgb(var(--gray-400-rgb) / <alpha-value>)', 500: 'rgb(var(--gray-500-rgb) / <alpha-value>)', 600: 'rgb(var(--gray-600-rgb) / <alpha-value>)', 700: 'rgb(var(--gray-700-rgb) / <alpha-value>)', 800: 'rgb(var(--gray-800-rgb) / <alpha-value>)', 900: 'rgb(var(--gray-900-rgb) / <alpha-value>)', 950: 'rgb(var(--gray-950-rgb) / <alpha-value>)' },
        zinc: { DEFAULT: 'var(--muted)', 50: 'rgb(var(--gray-50-rgb) / <alpha-value>)', 100: 'rgb(var(--gray-100-rgb) / <alpha-value>)', 200: 'rgb(var(--gray-200-rgb) / <alpha-value>)', 300: 'rgb(var(--gray-300-rgb) / <alpha-value>)', 400: 'rgb(var(--gray-400-rgb) / <alpha-value>)', 500: 'rgb(var(--gray-500-rgb) / <alpha-value>)', 600: 'rgb(var(--gray-600-rgb) / <alpha-value>)', 700: 'rgb(var(--gray-700-rgb) / <alpha-value>)', 800: 'rgb(var(--gray-800-rgb) / <alpha-value>)', 900: 'rgb(var(--gray-900-rgb) / <alpha-value>)', 950: 'rgb(var(--gray-950-rgb) / <alpha-value>)' },
        neutral: { DEFAULT: 'var(--muted)', 50: 'rgb(var(--gray-50-rgb) / <alpha-value>)', 100: 'rgb(var(--gray-100-rgb) / <alpha-value>)', 200: 'rgb(var(--gray-200-rgb) / <alpha-value>)', 300: 'rgb(var(--gray-300-rgb) / <alpha-value>)', 400: 'rgb(var(--gray-400-rgb) / <alpha-value>)', 500: 'rgb(var(--gray-500-rgb) / <alpha-value>)', 600: 'rgb(var(--gray-600-rgb) / <alpha-value>)', 700: 'rgb(var(--gray-700-rgb) / <alpha-value>)', 800: 'rgb(var(--gray-800-rgb) / <alpha-value>)', 900: 'rgb(var(--gray-900-rgb) / <alpha-value>)', 950: 'rgb(var(--gray-950-rgb) / <alpha-value>)' },
        stone: { DEFAULT: 'var(--muted)', 50: 'rgb(var(--surface-muted-rgb) / <alpha-value>)', 100: 'rgb(var(--gray-100-rgb) / <alpha-value>)', 200: 'rgb(var(--gray-200-rgb) / <alpha-value>)', 300: 'rgb(var(--gray-300-rgb) / <alpha-value>)', 400: 'rgb(var(--gray-400-rgb) / <alpha-value>)', 500: 'rgb(var(--gray-500-rgb) / <alpha-value>)', 600: 'rgb(var(--gray-600-rgb) / <alpha-value>)', 700: 'rgb(var(--gray-700-rgb) / <alpha-value>)', 800: 'rgb(var(--gray-800-rgb) / <alpha-value>)', 900: 'rgb(var(--gray-900-rgb) / <alpha-value>)', 950: 'rgb(var(--gray-950-rgb) / <alpha-value>)' },
        white: '#fff',
        black: '#000',
        red: { DEFAULT: 'var(--status-danger)', 50: 'rgb(var(--status-danger-rgb) / .08)', 100: 'rgb(var(--status-danger-rgb) / .14)', 200: 'rgb(var(--status-danger-rgb) / .22)', 300: 'rgb(var(--status-danger-rgb) / .32)', 400: 'rgb(var(--status-danger-rgb) / .48)', 500: 'rgb(var(--status-danger-rgb) / <alpha-value>)', 600: 'rgb(var(--status-danger-rgb) / <alpha-value>)', 700: 'rgb(var(--status-danger-rgb) / <alpha-value>)', 800: 'rgb(var(--status-danger-rgb) / <alpha-value>)', 900: 'rgb(var(--status-danger-rgb) / <alpha-value>)', 950: 'rgb(var(--status-danger-rgb) / <alpha-value>)' },
        orange: { DEFAULT: 'var(--status-warning)', 50: 'rgb(var(--status-warning-rgb) / .10)', 100: 'rgb(var(--status-warning-rgb) / .16)', 200: 'rgb(var(--status-warning-rgb) / .26)', 300: 'rgb(var(--status-warning-rgb) / .38)', 400: 'rgb(var(--status-warning-rgb) / .55)', 500: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 600: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 700: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 800: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 900: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 950: 'rgb(var(--status-warning-rgb) / <alpha-value>)' },
        amber: { DEFAULT: 'var(--status-warning)', 50: 'rgb(var(--status-warning-rgb) / .12)', 100: 'rgb(var(--status-warning-rgb) / .20)', 200: 'rgb(var(--status-warning-rgb) / .30)', 300: 'rgb(var(--status-warning-rgb) / .45)', 400: 'rgb(var(--status-warning-rgb) / .60)', 500: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 600: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 700: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 800: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 900: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 950: 'rgb(var(--status-warning-rgb) / <alpha-value>)' },
        yellow: { DEFAULT: 'var(--status-warning)', 50: 'rgb(var(--status-warning-rgb) / .12)', 100: 'rgb(var(--status-warning-rgb) / .20)', 200: 'rgb(var(--status-warning-rgb) / .30)', 300: 'rgb(var(--status-warning-rgb) / .45)', 400: 'rgb(var(--status-warning-rgb) / .60)', 500: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 600: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 700: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 800: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 900: 'rgb(var(--status-warning-rgb) / <alpha-value>)', 950: 'rgb(var(--status-warning-rgb) / <alpha-value>)' },
        green: { DEFAULT: 'var(--status-success)', 50: 'rgb(var(--status-success-rgb) / .10)', 100: 'rgb(var(--status-success-rgb) / .18)', 200: 'rgb(var(--status-success-rgb) / .28)', 300: 'rgb(var(--status-success-rgb) / .42)', 400: 'rgb(var(--status-success-rgb) / .58)', 500: 'rgb(var(--status-success-rgb) / <alpha-value>)', 600: 'rgb(var(--status-success-rgb) / <alpha-value>)', 700: 'rgb(var(--status-success-rgb) / <alpha-value>)', 800: 'rgb(var(--status-success-rgb) / <alpha-value>)', 900: 'rgb(var(--status-success-rgb) / <alpha-value>)', 950: 'rgb(var(--status-success-rgb) / <alpha-value>)' },
        emerald: { DEFAULT: 'var(--status-success)', 50: 'rgb(var(--status-success-rgb) / .10)', 100: 'rgb(var(--status-success-rgb) / .18)', 200: 'rgb(var(--status-success-rgb) / .28)', 300: 'rgb(var(--status-success-rgb) / .42)', 400: 'rgb(var(--status-success-rgb) / .58)', 500: 'rgb(var(--status-success-rgb) / <alpha-value>)', 600: 'rgb(var(--status-success-rgb) / <alpha-value>)', 700: 'rgb(var(--status-success-rgb) / <alpha-value>)', 800: 'rgb(var(--status-success-rgb) / <alpha-value>)', 900: 'rgb(var(--status-success-rgb) / <alpha-value>)', 950: 'rgb(var(--status-success-rgb) / <alpha-value>)' },
        'status-success': 'var(--status-success)',
        'status-warning': 'var(--status-warning)',
        'status-danger': 'var(--status-danger)'
      },
      fontFamily: {
        sans: ['system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: 'var(--shadow-glass)',
        'glass-sm': 'var(--shadow-glass-sm)',
        glow: 'var(--glow)',
        'glow-lg': 'var(--glow-lg)',
        card: 'var(--shadow-card)',
        'card-hover': 'var(--shadow-card-hover)',
        'inner-glow': 'var(--inner-glow)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, var(--primary-500), var(--primary-700))',
        'gradient-dark': 'linear-gradient(135deg, var(--dark-800), var(--dark-950))',
        'gradient-glass': 'linear-gradient(135deg, var(--glass-tint), var(--glass-tint-strong))',
        'mesh-gradient': 'radial-gradient(at 40% 20%, var(--mesh-a) 0px, transparent 50%), radial-gradient(at 80% 0%, var(--mesh-b) 0px, transparent 50%), radial-gradient(at 0% 50%, var(--mesh-c) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: { '0%': { opacity: '0' }, '100%': { opacity: '1' } },
        slideUp: { '0%': { opacity: '0', transform: 'translateY(10px)' }, '100%': { opacity: '1', transform: 'translateY(0)' } },
        slideDown: { '0%': { opacity: '0', transform: 'translateY(-10px)' }, '100%': { opacity: '1', transform: 'translateY(0)' } },
        slideInRight: { '0%': { opacity: '0', transform: 'translateX(20px)' }, '100%': { opacity: '1', transform: 'translateX(0)' } },
        scaleIn: { '0%': { opacity: '0', transform: 'scale(0.95)' }, '100%': { opacity: '1', transform: 'scale(1)' } },
        shimmer: { '0%': { backgroundPosition: '-200% 0' }, '100%': { backgroundPosition: '200% 0' } },
        glow: { '0%': { boxShadow: 'var(--glow)' }, '100%': { boxShadow: 'var(--glow-lg)' } }
      },
      backdropBlur: { xs: '2px' },
      borderRadius: {
        DEFAULT: 'var(--radius-md)',
        sm: 'var(--radius-sm)',
        md: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
        xl: 'var(--radius-xl)',
        '2xl': 'var(--radius-2xl)',
        '3xl': 'var(--radius-3xl)',
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}

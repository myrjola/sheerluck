const defaultTheme = require("tailwindcss/defaultTheme");

/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./ui/**/*.{html,js,gohtml,templ}"],
    theme: {
        extend: {
            fontFamily: {
                sans: ["Inter var", ...defaultTheme.fontFamily.sans],
            },
            animation: {
                'fade-in': 'fade-in 0.7s ease-out',
                'fade-out': 'fade-out 0.7s ease-out',
                'slide-in': 'slide-in 0.3s ease-in-out',
                'slide-out': 'slide-out 0.3s ease-in-out',
            },
            keyframes: {
                'fade-in': {
                    '0%': {
                        opacity: '0',
                    },
                    '100%': {
                        opacity: '1',
                    },
                },
                'fade-out': {
                    '0%': {
                        opacity: '1',
                    },
                    '100%': {
                        opacity: '0',
                    },
                },
                'slide-in': {
                    '0%': {
                        'transform': 'translateX(-100%)',
                    },
                    '100%': {
                        'transform': 'translateX(0)',
                    },
                },
                'slide-out': {
                    '0%': {
                        'transform': 'translateX(0)',
                    },
                    '100%': {
                        'transform': 'translateX(-100%)',
                    },
                }
            }
        },
    },
    plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};

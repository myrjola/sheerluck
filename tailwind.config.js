const defaultTheme = require("tailwindcss/defaultTheme");

/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./ui/html/**/*.{html,js,gohtml,templ}"],
    theme: {
        extend: {
            fontFamily: {
                sans: ["Inter var", ...defaultTheme.fontFamily.sans],
            },
            animation: {
                'fade-in': 'fade-in 0.7s ease-out',
                'fade-out': 'fade-out 0.7s ease-out'
            },
            keyframes: {
                "fade-in": {
                    '0%': {
                        opacity: '0',
                        display: 'none'
                    },
                    '100%': {
                        opacity: '1',
                        display: 'block'
                    },
                },
                "fade-out": {
                    '0%': {
                        opacity: '1',
                        display: 'block'
                    },
                    '100%': {
                        opacity: '0',
                        display: 'none'
                    },
                }
            }
        },
    },
    plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};

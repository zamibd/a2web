module.exports = {
    content: ["./web/templates/*.html", "./web/templates/**/*.html"],
    theme: {
        extend: {},
    },
    plugins: [require("daisyui")],
    daisyui: {
        themes: ["dark"], // Force dark mode as per layout
    },
}

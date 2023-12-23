# sheerluck

AI-powered murder mystery game

# Quickstart

## TailwindCSS

### Standalone executable

This project uses [tailwindcss](https://tailwindcss.com/). You can use it without setting up Node.js using [standalone executables](https://tailwindcss.com/blog/standalone-cli).

Download latest [tailwindcss executable for your platform](https://github.com/tailwindlabs/tailwindcss/releases/tag/v3.3.5).

Start watching for changes in the templates. This generates ui/static/main.css loaded at every page.

```
./tailwindcss -i input.css -o ui/static/main.css --watch
```

### Jetbrains autocomplete for TailwindCSS and prettier class sorter

Unfortunately, [Jetbrains Tailwind CSS plugin](https://www.jetbrains.com/help/webstorm/tailwind-css.html) may not support the standalone Tailwind executable. To get around this, you have to initialize the NodeJs project and maybe restart the IDE.

```
nvm use
pnpm i
```

## Generate templ code

This project uses [Templ](https://templ.guide/) for the HTML templates.

When you do changes to the `.templ` files, you need to run the following:

```

```


## Start go server

Make sure you're using the go version configured in `go.mod`. To start the server, run the following:

```
go run ./cmd/web/
```

Navigate to http://localhost:3003 to see the service in action.

### Acknowledgements

Images was created with the assistance of DALL·E 2
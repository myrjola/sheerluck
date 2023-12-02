# sheerluck

AI-powered murder mystery game

# Quickstart

## TailwindCSS

This project uses [tailwindcss](https://github.com/tailwindlabs/tailwindcss/releases/tag/v3.3.5) 

Download latest [tailwindcss executable for your platform](https://github.com/tailwindlabs/tailwindcss/releases/tag/v3.3.5).

Start watching for changes in the templates. This generates ui/static/main.css loaded at every page.

```
./tailwindcss -i input.css -o ui/static/main.css --watch
```

## Start go server

Make sure you're using the go version configured in `go.mod`. To start the server, run the following:

```
go run ./cmd/web/
```

Navigate to http://localhost:3003 to see the service in action.


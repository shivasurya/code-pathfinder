### Code-Pathfinder Playground

The Code-Pathfinder Playground is a online interactive app that allows you to analyze code and execute Code-Pathfinder (CodeQL) queries on it.

![Code-Pathfinder Playground](https://badgen.net/static/Online%20Playground/live/cyan?icon=terminal)

### Quickstart

In the playground directory, run:

```shell
$ go run main.go
```

This will start the playground server. Visit `http://localhost:8080` to access the playground.

### Docker Build

From the root directory, run:

```shell
$ podman build --platform linux/amd64 -t docker.io/shivasurya/cpf-playground:latest . -f playground-Dockerfile
```

This will build the playground Docker image.


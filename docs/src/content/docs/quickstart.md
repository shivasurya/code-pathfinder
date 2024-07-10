---
title: Quickstart
description: "A quickstart guide to get started with Code-PathFinder."
---

## Installation

You can install Code-PathFinder using pre-built binaries from [GitHub releases](https://github.com/shivasurya/code-pathfinder/releases) or from source.
More installation support via homebrew, npm, curl will be added soon.

### Pre-Built Binaries

Download the latest release from [GitHub releases](https://github.com/shivasurya/code-pathfinder/releases) and choose 
the binary that matches your operating system.

```shell
$ chmod u+x pathfinder
$ pathfinder --help
```

### From Source

Ensure you have [Gradle](https://gradle.org/) and [GoLang](https://go.dev/doc/install) installed.

```shell
$ git clone https://github.com/shivasurya/code-pathfinder
$ cd sourcecode-parser
$ gradle buildGo
$ build/go/pathfinder --help
```

### Sanity Check

Check if Code-PathFinder is working properly by running the following command:

```shell
$ pathfinder --version
Version: 0.0.16
Git Commit: 40886e7
```

```shell
$ pathfinder --help
Usage of pathfinder:
  -output string
        Supported output format: json
  -output-file string
        Output file path
  -project string
        Project to analyze
  -query string
        Query to execute
  -stdin
        Read query from stdin
  -version
        Print the version information and exit
```

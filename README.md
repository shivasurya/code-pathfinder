<p>
<div align="center">
  <img src="./assets/cpv.png" alt="Code Pathfinder" width="100" height="100"/>
</p>

# Code Pathfinder 
About
Code Pathfinder, the open-source alternative to GitHub CodeQL. Built for advanced structural search, derive insights, find vulnerabilities in code.

[![Build and Release](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml/badge.svg)](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/shivasurya/code-pathfinder/sourcecode-parser)](https://goreportcard.com/report/github.com/shivasurya/code-pathfinder/sourcecode-parser)
[![MIT License](https://img.shields.io/github/license/shivasurya/code-pathfinder)](https://github.com/shivasurya/code-pathfinder/blob/main/LICENSE)
[![Discord](https://img.shields.io/discord/1259511338183557120?logo=discord&label=discord&utm_source=github)](https://discord.gg/xmPdJC6WPX)
[![codecov](https://codecov.io/gh/shivasurya/code-pathfinder/graph/badge.svg?token=VYQLI49TF4)](https://codecov.io/gh/shivasurya/code-pathfinder)
</div>

## :tv: Demo

```bash
docker run --rm -v "./src:/src" shivasurya/code-pathfinder:stable-latest ci --project /src/code-pathfinder/test-src --ruleset cpf/java
```

## :book: Documentation

- [Documentation](https://codepathfinder.dev/)
- [Pathfinder Queries](https://github.com/shivasurya/code-pathfinder/tree/main/pathfinder-rules)


## :floppy_disk: Installation

### :whale: Using Docker

```bash
$ docker pull shivasurya/code-pathfinder:stable-latest
```

### From npm

```bash
$ npm install -g codepathfinder
$ pathfinder --help
```

### Pre-Built Binaries

Download the latest release from [GitHub releases](https://github.com/shivasurya/code-pathfinder/releases) and choose
the binary that matches your operating system.

```shell
$ chmod u+x pathfinder
$ pathfinder --help
```


## Getting Started
Read the [official documentation](https://codepathfinder.dev/), or run `pathfinder --help`.

## Features

- [x] Basic queries (Similar to CodeQL)
- [x] Source Sink Analysis
- [ ] Data Flow Analysis with Control Flow Graph

## Usage

```bash
$ cd sourcecode-parser

$ gradle buildGo (or) npm install -g codepathfinder

$ ./pathfinder query --project <path_to_project> --stdin
2024/06/30 21:35:29 Graph built successfully
Path-Finder Query Console: 
>FROM method_declaration AS md 
 WHERE md.getName() == "getPaneChanges"
 SELECT md, "query for pane changes layout methods"
Executing query: FROM method_declaration AS md WHERE md.getName() == "getPaneChanges"

┌───┬──────────────────────────────────────────┬─────────────┬────────────────────┬────────────────┬──────────────────────────────────────────────────────────────┐
│ # │ FILE                                     │ LINE NUMBER │ TYPE               │ NAME           │ CODE SNIPPET                                                 │
├───┼──────────────────────────────────────────┼─────────────┼────────────────────┼────────────────┼──────────────────────────────────────────────────────────────┤
│ 1 │ /Users/shiva/src/code-pathfinder/test-sr │         148 │ method_declaration │ getPaneChanges │ protected void getPaneChanges() throws ClassCastException {  │
│   │ c/android/app/src/main/java/com/ivb/udac │             │                    │                │         mTwoPane = findViewById(R.id.movie_detail_container) │
│   │ ity/movieListActivity.java               │             │                    │                │  != null;                                                    │
│   │                                          │             │                    │                │     }                                                        │
└───┴──────────────────────────────────────────┴─────────────┴────────────────────┴────────────────┴──────────────────────────────────────────────────────────────┘
Path-Finder Query Console: 
>:quit
Okay, Bye!
```

## Acknowledgements
Code Pathfinder uses tree-sitter for all language parsers.


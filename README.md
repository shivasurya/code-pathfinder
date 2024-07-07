<p>
<div align="center">
  <img src="./assets/cpv.png" alt="Code Pathfinder" width="100" height="100"/>
</p>

# Code Pathfinder 
Code Pathfinder attempts to be query language for structural search on source code. It's built for identifying vulnerabilities in source code. Currently, it only supports Java language.

[![Build and Release](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml/badge.svg)](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/shivasurya/code-pathfinder/sourcecode-parser)](https://goreportcard.com/report/github.com/shivasurya/code-pathfinder/sourcecode-parser)
[![MIT License](https://img.shields.io/github/license/shivasurya/code-pathfinder)](https://github.com/shivasurya/code-pathfinder/blob/main/LICENSE)
[![Discord](https://img.shields.io/discord/1259511338183557120?logo=discord&label=discord&utm_source=github)](https://discord.gg/xmPdJC6WPX)
</div>

## Getting Started
Read the [official documentation](https://codepathfinder.dev/), or run `pathfinder --help`.

## Features

- [x] Basic queries
- [x] Source Sink Analysis
- [ ] Taint Analysis
- [ ] Data Flow Analysis with Control Flow Graph

## Usage

```bash
$ cd sourcecode-parser

$ go build -o pathfinder (or) go run .

$ ./pathfinder /PATH/TO/SOURCE
2024/06/30 21:35:29 Graph built successfully
Path-Finder Query Console: 
>FIND method_declaration WHERE throwstype = "ClassCastException"
Executing query: FIND method_declaration WHERE throwstype = "ClassCastException"

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


<p>
<div align="center">
  <img src="./assets/cpv.png" alt="Code Pathfinder" width="100" height="100"/>
</p>


[![Build](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml/badge.svg)](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml)
[![VS Code Marketplace](https://img.shields.io/visual-studio-marketplace/v/codepathfinder.secureflow?label=VS%20Code&logo=visualstudiocode)](https://marketplace.visualstudio.com/items?itemName=codepathfinder.secureflow)
[![npm version](https://img.shields.io/npm/v/@codepathfinder/secureflow-cli?logo=npm)](https://www.npmjs.com/package/@codepathfinder/secureflow-cli)
[![Open VSX](https://img.shields.io/open-vsx/v/codepathfinder/secureflow?label=Open%20VSX&logo=vscodium)](https://open-vsx.org/extension/codepathfinder/secureflow)
[![AGPL-3.0 License](https://img.shields.io/github/license/shivasurya/code-pathfinder)](https://github.com/shivasurya/code-pathfinder/blob/main/LICENSE)
</div>

# Code Pathfinder 

**An open-source security suite aiming to combine structural code analysis with AI-powered vulnerability detection.**

Code Pathfinder is designed to bridge the gap between traditional static analysis tools and modern AI-assisted security review. While mature tools excel at pattern matching and complex queries, Code Pathfinder focuses on making security analysis more accessible, leveraging large language models to understand context and identify nuanced vulnerabilities, and integrated throughout the development lifecycle.

- **Real-time IDE integration** - Bringing security insights directly into your editor as you code
- **AI-assisted analysis** - Leveraging large language models to understand context and identify nuanced vulnerabilities
- **Unified workflow coverage** - From local development to pull requests to CI/CD pipelines
- **Flexible reporting** - Supporting DefectDojo, GitHub Advanced Security, SARIF, and other platforms

Built for security engineers and developers who want an extensible, open-source alternative that's evolving with modern development practices. Here are the initiatives:

- **[Code-Pathfinder CLI](https://github.com/shivasurya/code-pathfinder/releases)** - Basic security analysis scanner better than grep
- **[Secureflow CLI](https://github.com/shivasurya/code-pathfinder/tree/main/extension/secureflow/packages/secureflow-cli)** - Claude-Code for security analysis powered by large language models with better context engineering
- **[Secureflow VSCode Extension](https://github.com/shivasurya/code-pathfinder/tree/main/extension/secureflow)** - IDE integration for security analysis powered by large language models with better context engineering

## :tv: Demo

### Secureflow CLI

```bash
$ secureflow scan ./path/to/project
```

### Code-Pathfinder CLI

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

#### Secureflow CLI

```bash
$ npm install -g @codepathfinder/secureflow-cli
$ secureflow scan --help
```

#### Code-Pathfinder CLI

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


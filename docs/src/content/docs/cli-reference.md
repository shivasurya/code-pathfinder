---
title: CLI Reference
description: "Code PathFinder CLI reference guide"
---

This guide provides a comprehensive list of all command-line options and flags available in Code PathFinder.

## Basic Syntax

```shell
pathfinder --project <project_dir> <flags>
```

## Flags

| Option                    | Description                                              |
|---------------------------|----------------------------------------------------------|
| `--project <project_dir>` | Path to project directory to run analysis                |
| `---query "TYPE"`         | Query to execute on the project                          |
| `--output TYPE`           | Output format. json is the only supported format for now |
| `--output-file FILE.json` | Output file path with name to write the result.          |
| `--stdin true`            | Launch stdin query console interactive way to query      |
| `--version`               | Print installed version and commit tag information       |
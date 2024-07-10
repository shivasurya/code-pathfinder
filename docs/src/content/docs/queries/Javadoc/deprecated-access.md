---
title: Deprecated Method Query
description: "Code PathFinder Query for finding deprecated method access"
---

## Avoid Deprecated Callable Access Query

Pathfinder supports querying for deprecated callable access in the source code. The query is designed to identify deprecated callable access in the source code.

## Query Syntax

The query syntax is simple, easy to use and inspired by SQL. The has_access routine is used to check if the callable is being accessed across the project.

```sql
FIND method_declaration WHERE annotation = "@Deprecated" AND has_access = "true"
```
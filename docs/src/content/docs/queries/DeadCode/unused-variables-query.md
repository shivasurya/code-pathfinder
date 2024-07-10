---
title: Unused Variables Query
description: "Code PathFinder Query for finding unused variables"
---

## Unused Variables Query

Pathfinder supports querying for unused variables in the source code. The query is designed to identify variables that are declared but not used in the source code.

## Query Syntax

The query syntax is simple, easy to use and inspired by SQL. The has_access routine is used to check if the variable is being accessed based on the scope.

```sql
FIND variable_declaration WHERE has_access = false AND scope = 'local'
```
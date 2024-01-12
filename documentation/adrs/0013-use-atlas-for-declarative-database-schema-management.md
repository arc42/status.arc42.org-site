# 13. use Atlas for declarative database schema management

Date: 2024-01-06

## Status

Accepted

## Context

* We will evolve the database schema in course of development.
* Manual DB migration or schema evolution is a nightmare, therefore we want an automated solution


## Decision

[Atlas](https://atlasgo.io) is a free solution which enables declarative schema management.

A [blogpost on turso.tech](https://blog.turso.tech/database-migrations-made-easy-with-atlas-df2b259862db) describes some more details.

## Consequences

* describe the (desired) DB schema in HCL (the language invented by HashiCorp for Terraform), file "db-schema.hcl"
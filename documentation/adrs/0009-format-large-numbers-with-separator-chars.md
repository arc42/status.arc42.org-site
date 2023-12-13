# 9. format large numbers with separator chars

Date: 2023-12-12

## Status

Accepted

## Context

The (fairly) large access numbers for the sites result in numbers difficult to read (e.g. 805455).

## Decision

Numbers in the generated HTML table shall be formatted using separators, resulting in e.g. 805.455

We decided to use the following way to get separators:

---
p := message.NewPrinter(language.German)
myStr := p.Sprintf( "%d", 1234567)
// myStr == "1.234.567"

## Consequences

* Some types (e.g. types/TotalsForAllSites) need to carry around both an int PLUS a string representation of the same value.
The string is formatted with decimal separators, whereas the numbers aren't.

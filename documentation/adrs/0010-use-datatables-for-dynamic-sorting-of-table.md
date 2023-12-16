# 10. use DataTables for dynamic sorting of table

Date: 2023-12-16

## Status

Accepted

## Context

Users (in browser) shall be able to sort the table by the various columns - see [issue #69](https://github.com/arc42/status.arc42.org-site/issues/69).

## Decision

* To make sorting dynamic, we use JavaScript for sorting
* We use the [DataTables](https://datatables.net/manual/options) library for sorting, that's based upon JQuery.

## Consequences

We need to:

* download the respective JS libraries, as documented in their installation guide
* download their css
* add the required boilerplate to the gohtml template
  * <thead></thead>
  * <tbody></tbody>
  * <tfoot></tfoot>
  * set table attributes: `<table id="statsTable" class="display">
    `  
* create a sample HTMl for demo and testing
# Local JavaScript/CSS Dependencies

This directory contains local copies of third-party JavaScript and CSS libraries to avoid CDN dependencies.

## Files included:

### jQuery 3.5.1
- **File**: `jquery-3.5.1.min.js`
- **Source**: https://code.jquery.com/jquery-3.5.1.min.js
- **Purpose**: Required for DataTables functionality

### DataTables 1.13.7
- **Files**: 
  - `jquery.dataTables.min.js`
  - `css/jquery.dataTables.min.css`
- **Source**: 
  - JS: https://cdn.datatables.net/1.13.7/js/jquery.dataTables.js
  - CSS: https://cdn.datatables.net/1.13.7/css/jquery.dataTables.css
- **Purpose**: Provides sortable table functionality for the statistics tables

## Note
The current files are placeholders. For production use, download the actual library files from the URLs above and replace the placeholder content.

## Usage
These libraries are included in the site via `docs/_includes/head/custom.html` and are used to enable dynamic table sorting on the main statistics page.
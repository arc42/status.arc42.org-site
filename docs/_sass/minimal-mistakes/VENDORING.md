# Vendored theme: minimal-mistakes 4.24.0

**Baseline:** [mmistakes/minimal-mistakes](https://github.com/mmistakes/minimal-mistakes)
tag `4.24.0`, commit `00fa7be38b5e57c6539d3986b2168ea7b57bc67a` (2021-07-05).
**Vendored:** 2026-08-04.
**Licence:** MIT — the theme's own `LICENSE`, unchanged, sits next to this file.

## Why

`_config.yml` used to carry `remote_theme: "mmistakes/minimal-mistakes@4.24.0"`.
That is a network call on every build: local `make site`, the GitHub Actions
job, and anybody's first clone all had to reach github.com before Jekyll could
render a page, and the build broke the day GitHub was slow or the tag moved.
The theme is a frozen 2021 release that nobody here upgrades casually, so the
honest thing is to keep it in the repository and read it from disk.

## What was copied

From the tag, verbatim:

| Source | Destination |
|---|---|
| `_layouts/` | `docs/_layouts/` |
| `_includes/` | `docs/_includes/` |
| `_sass/` | `docs/_sass/` |
| `assets/js/` | `docs/assets/js/` |

`assets/css/main.scss` was **not** copied — this site already had its own, and
it is the file that binds the family type stack to the theme's variables.

`assets/js/` is copied even though `_layouts/default.html` deliberately does not
include `scripts.html` (the theme's scripts fight DataTables). Without it the
vendored tree would be a theme that cannot be turned back on offline; with it,
re-enabling `scripts.html` is a one-line change and still needs no network.

The theme's `_data/` (`ui-text.yml`, `navigation.yml`) was **not** copied, and
that is not an omission: Jekyll themes never supplied `_data` to a site, so the
site never had it under `remote_theme` either. Every template reads it as
`site.data.ui-text[site.locale].x | default: "X"` and falls through to the
default. Adding `_data/ui-text.yml` later is a safe, separate change.

## Local overrides — these win, do not overwrite them on an upgrade

Files that exist in both trees, where **the local version is the one in force**:

| File | Why it differs |
|---|---|
| `_layouts/default.html` | omits `scripts.html`; the theme's jQuery plugins conflict with the DataTables setup this site loads in `_includes/head/custom.html` |
| `_includes/masthead.html` | family band + `arc42-logo-white.svg`, no nav lists, no search toggle |
| `_includes/footer.html` | the arc42 family footer (ported from docs.arc42.org) instead of the theme's social-profile row |
| `_includes/head/custom.html` | plausible, htmx, DataTables, favicons — never part of the theme |
| `assets/css/main.scss` | sets the family type variables *before* importing the theme, then loads the site's own partials |

## Deliberate edits inside the vendored tree

Kept to the minimum, each marked in place with a comment starting
`arc42 change to the vendored theme`:

1. `_includes/head.html` — removed the FontAwesome 5 `preload`/`noscript` pair
   pointing at `cdn.jsdelivr.net`. A site with an imprint does not hand a
   visitor's IP to a third-party asset host.
2. Every `<i class="fa…">` in `_includes/` and `_layouts/` — removed. All of
   them sat next to a visible text label, which is what the whole arc42 family
   settled on for footers and bylines on 2026-07-30. The two exceptions:
   `archive-single.html`'s permalink glyph became inline SVG via
   `_includes/icon.html`, and the comment-form spinners became
   `<span class="loading-spinner">` (styled in `_sass/_arc42-family.scss`).
3. `_sass/minimal-mistakes/_utilities.scss` — dropped the 24 `.social-icons
   .fa-*` brand-colour rules, which addressed classes that no longer exist.
   The `$github-color`-style variables they used are still in `_variables.scss`.
4. `assets/js/_main.js` **and** `assets/js/main.min.js` — the heading-permalink
   anchor injected `<i class="fas fa-link">` at runtime. Both copies now inject
   the inline-SVG `<use>` instead, so the file cannot reintroduce the
   dependency the day somebody re-enables `scripts.html`.

Everything else is byte-identical to the tag.

Two files under `assets/` are **not** from the theme and are not part of this
baseline: `assets/icons/ui.svg` and `_includes/icon.html`, both ported from
docs.arc42.org-site. They are the family's icon mechanism and the replacement
for everything FontAwesome used to draw.

## Upgrading

1. `git clone --depth 1 --branch <newtag> https://github.com/mmistakes/minimal-mistakes`
   into a scratch directory outside this repo.
2. Diff that tree against this one to see what upstream changed *and* to
   re-apply the three edits above.
3. Restore the five local overrides listed in the table — `git checkout --` on
   those paths after copying is enough.
4. Re-run the offline check: build, and confirm the log has no `Remote Theme`
   line and the rendered page makes no request to a font or icon CDN.

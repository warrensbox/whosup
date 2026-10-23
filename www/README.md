# whosup website

The source for [whosup.warrensbox.com](https://whosup.warrensbox.com), built with [Zensical](https://zensical.org).

```text
www/
├── zensical.toml   # site settings and navigation
├── docs/           # the pages, in Markdown
└── site/           # build output (not committed)
```

## Set up

You need Python 3. Run all commands from this `www/` folder.

```sh
cd www
python3 -m venv .venv
source .venv/bin/activate
pip install zensical==0.0.71
```

Use the same Zensical version as the release workflow (`.github/workflows/release.yml`), so what you see locally matches what gets published.

## Preview your changes

```sh
zensical serve
```

Open <http://localhost:8000>. The page reloads whenever you save a file.

## Check before you commit

```sh
zensical build --clean --strict
```

This is the same build the release workflow runs. `--strict` fails on warnings, such as a link to a page that doesn't exist, so fix anything it reports. It doesn't catch a page you forgot to add to `nav`, so check the menu in the preview.

## Add or edit a page

- Edit an existing page in `docs/`.
- To add a page, create a Markdown file in `docs/` and add it to `nav` in `zensical.toml`, or it won't appear in the menu:

  ```toml
  nav = [
    { "Home" = "index.md" },
    { "Installation" = "installation.md" },
    { "Usage" = "usage.md" },
    { "Your page" = "your-page.md" },
  ]
  ```

- Link between pages with relative paths, like `[Usage](usage.md)`.

## Publishing

You don't publish the site by hand. When a release runs (Actions → **Release (Manual Step)**), the `docs-deploy` job builds this folder and publishes it to GitHub Pages, after the release itself succeeds. Changes merged to `main` go live with the next release.

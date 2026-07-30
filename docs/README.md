# Documentation

This folder contains the [VitePress](https://vitepress.dev/) site published to
[foomo.github.io/contentserver](https://foomo.github.io/contentserver/).

## Local development

Dependencies are managed with [Bun](https://bun.sh/).

```bash
cd docs
bun install --frozen-lockfile

bun run dev      # start the dev server with hot reload
bun run build    # build the static site into .vitepress/dist
bun run preview  # preview the production build locally
```

## Publishing

Publication is automated by [`.github/workflows/docs.yml`](../.github/workflows/docs.yml).
The workflow builds the site with Bun and deploys it to GitHub Pages. It runs on:

- a push of a version tag matching `v*.*.*`, and
- manual dispatch (**Actions → Publish docs → Run workflow**).

## One-time GitHub repository configuration

The workflow already requests the required `pages: write` and `id-token: write`
permissions, but GitHub Pages must be switched to the GitHub Actions source once
per repository before the site becomes visible:

1. Go to **Settings → Pages**.
2. Under **Build and deployment → Source**, select **GitHub Actions**
   (not "Deploy from a branch").
3. (Optional) Under **Settings → Environments → github-pages**, review the
   deployment protection rules. The `docs.yml` workflow deploys to the
   `github-pages` environment; by default any protection rules there apply.
4. Trigger a first deployment — either push a `vX.Y.Z` tag or run the
   **Publish docs** workflow manually via **Actions → Publish docs → Run workflow**.
5. Once the workflow succeeds, the site is available at
   <https://foomo.github.io/contentserver/>. The URL is also shown on the completed
   workflow run's **github-pages** environment.

No other repository settings are required — the `base` path (`/contentserver/`) and
sitemap hostname are already configured in
[`.vitepress/config.mts`](.vitepress/config.mts).

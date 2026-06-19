# nais-example

A minimal, opinionated example of how to structure a repository for a single
[Nais](https://nais.io) application that owns a
[Valkey](https://docs.nais.io/persistence/valkey/) cache. It runs in two
environments (`dev-gcp` and `prod-gcp`).

This repo is meant as a golden example: minimalistic, but showing the patterns
we want teams to copy.

## What the app does

`cmd/server` is a tiny HTTP server that prints a greeting on `/` and serves a
health check on `/healthz`. The Valkey connection details are injected by Nais.
That's it — the point is the **structure**, not the app.

## Layout

```
.nais/
  app.yaml             # Application (base = dev-gcp) — deployed by deploy-app
  app.prod-gcp.yaml    # per-env mixin (auto-resolved)
  valkey.yaml          # Valkey (base = dev-gcp) — deployed by deploy-nais-resources
  valkey.prod-gcp.yaml # per-env mixin
cmd/server/main.go     # the application
Dockerfile
```

All Nais manifests live under `.nais/`.

## Mixins and explicit per-environment config

Each manifest has a base file plus an optional per-environment **mixin** named
`<base>.<environment>.yaml`. The base holds the **dev-gcp** config; when you run
`nais alpha apply <base>.yaml --environment <env>`, the CLI automatically
deep-merges the matching mixin over the base (mixin values win). Keeping dev as
the base means overrides only ever scale *up* for prod, and a forgotten mixin
fails safe to the smaller dev config. The per-environment differences stay
explicit and small:

- **app**: `prod-gcp` gets extra replicas and marginally more CPU/memory.
- **valkey**: `dev-gcp` runs `SingleNode`/`1GB`; `prod-gcp` runs
  `HighAvailability`/`4GB`.

Note: the mixin merge **overrides** maps/scalars but **concatenates** lists, so
per-environment values belong in map fields (like `resources`), not in list
fields like `spec.env`.

## Two workflows, two responsibilities

We deliberately split deployment into two workflows so that source changes and
resource changes are deployed independently:

| Workflow | Triggers on | Does |
| --- | --- | --- |
| [`deploy-app`](.github/workflows/deploy-app.yaml) | source code (`cmd/**`, `go.*`, `Dockerfile`) | builds + pushes the image, then `nais apply app.yaml --set spec.image=<new image>` to each env |
| [`deploy-nais-resources`](.github/workflows/deploy-nais-resources.yaml) | any change under `.nais/**` | `nais apply` for Valkey and the app manifest to each env |
| [`cleanup-valkey`](.github/workflows/cleanup-valkey.yaml) | nightly schedule | `nais valkey delete` for each env |

Why split them?

- A source change builds a new image and rolls it out; a manifest change (app
  or Valkey) is applied without touching the image.
- The app manifest carries **no image**. `deploy-app` injects the freshly built
  tag with `--set spec.image`; `deploy-nais-resources` applies it without an
  image, so the CLI keeps whichever image is currently running (it queries the
  Nais API). The manifest stays free of image churn either way.
- First deploy of a brand-new app must come from `deploy-app`, since there is no
  running image for the CLI to reuse yet.
- Edge case: a commit touching both source and `.nais/app.yaml` triggers both
  workflows; the build run normally lands last (it's slower) and wins. Avoid
  mixing code and app-manifest changes in one commit.

Both workflows authenticate to Nais purely via GitHub's OIDC token
(`permissions: id-token: write`) and use [`nais/setup`](https://github.com/nais/setup)
to install the CLI — no long-lived secrets for deploying.

## Lifecycle / cleanup

This example is disposable — it cleans up after itself so it never lingers:

- The **application** sets `spec.ttl: "1h"`; the platform deletes it that long
  after each deploy. Bump or remove the field if you want it to stay up.
- The **Valkey** instance is a separate resource the app does not own, so the
  TTL does not reach it. The [`cleanup-valkey`](.github/workflows/cleanup-valkey.yaml)
  workflow deletes it on a nightly schedule (`nais valkey delete cache -y`).
  A Valkey can only be deleted once no workload references it, which the app's
  TTL ensures by then.

## Local development

```bash
go run ./cmd/server
# open http://localhost:8080
```

# nais-example

A minimal, opinionated example of how to structure a repository for a single [Nais](https://nais.io) application that owns a [Valkey](https://docs.nais.io/persistence/valkey/) cache. It runs in two environments (`dev-gcp` and `prod-gcp`).

## Layout

```
.nais/
  app.yaml             # Application (base = dev-gcp) — deployed by deploy-app
  app.prod-gcp.yaml    # production mixin (auto-resolved)
  valkey.yaml          # Valkey (base = dev-gcp) — deployed by deploy-nais-resources
  valkey.prod-gcp.yaml # production mixin
cmd/server/main.go     # the application
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

A source change builds a new image and rolls it out; a manifest change (app or Valkey) is applied without touching the image.
The app manifest carries **no image**. `deploy-app` injects the freshly built tag with `--set spec.image`; `deploy-nais-resources` applies it without an image, so the CLI keeps whichever image is currently running (it queries the Nais API).

# Changelog

## Unreleased (planned as v1.2.0)

### Breaking
- Terraform 1.11 or later is required. A certificate and its private key, a mac bare metal device's
  password and a server's `password` and `cloud_init` are write-only arguments, which earlier Terraform
  versions cannot set.

### New
- `flow_compute_snapshot` resource — implemented but never registered.
- `timeouts {}` on every resource that waits.
- `status` on `flow_compute_load_balancer_member` and its data source.
- `public` on `flow_compute_load_balancer` gives the load balancer a public ip, with the address in
  `public_ip`. The api only takes it at create, so changing it replaces the load balancer.
- `flow_compute_elastic_ip_load_balancer_attachment` attaches an elastic ip you manage to an existing
  load balancer, the same way `flow_compute_elastic_ip_server_attachment` does for a server.

### Fixes
- Resources the api created are no longer lost from the state when a later step fails.
- Values of `false` and `0` reach the api, so settings can be switched off again and ping rules work.
- Turning a router private gives up its public ip instead of failing the apply.
- State matches the api after an update.
- Imports no longer plan a replace, and secrets stay out of the state file.
- Mac bare metal resources are as reliable as compute: retries, and cleanup of deleted resources.
- A destroy waits until the api has really removed a load balancer, server, volume, snapshot or
  device instead of returning on the delete call. Destroying a network behind a load balancer no
  longer fails while the teardown is still running. `timeouts { delete }` bounds the wait.
- Load balancer pool updates and destroys are faster and no longer rebuild the health monitor needlessly.
- Clearer errors for kubernetes version changes and an empty token; several data source fixes.

### Dependencies
- framework v0.10.0 → v1.19.0, plugin-go v0.13.0 → v0.31.0, tests on plugin-testing v1.16.0; no
  behaviour change, generated docs byte-identical.
- New: terraform-plugin-framework-timeouts v0.7.0. CI matrix on the provider's minimum (1.11) and the latest Terraform release.

## Unreleased (planned as v1.1.3)

### Behaviour changes
- `flow_compute_server.product_id` resizes the server in place (stop, resize, start — about a minute
  of downtime) instead of replacing it.
- `flow_compute_volume_attachment.volume_id` replaces the attachment (detach, attach).
- `flow_compute_network_interface.security_group_ids` is a set instead of a list; existing configs
  and states keep working.
- `flow_compute_volume.name` is required — the api refuses a create without one, so the error moves
  from a failed apply to the plan.

### Reliability
- Mutating API calls are retried with a bounded backoff instead of failing on the first transient
  error (new provider attribute `retry_timeout`, default `90s`, `0` disables); reads are retried on
  gateway errors.
- The provider waits until resources are actually usable — servers running, volumes settled, load
  balancers mutable, clusters ready or gone — with a deadline and a clear error on every wait.
- A create that fails halfway leaves a tainted resource instead of an untracked one, and objects
  deleted outside Terraform are dropped from the state and recreated instead of failing every plan.
- A 404 counts as success only for deletes and detaches; on updates the api's error surfaces —
  the api answers 404 for missing sub-entities too (e.g. an unknown cluster version), which was
  silently mistaken for success before.

### Fixes and additions
- Kubernetes clusters can be updated without pinning `version_id`, and a version change keeps the
  cluster's configuration variables.
- Renames no longer rebuild dependent resources such as routes, pool members and elastic-IP
  attachments.
- `key_pair_id` and `network_id` may be omitted; the values the api assigns are adopted cleanly.
- `flow_compute_server` exposes `network_interface_id` and `security_group_ids`: security groups on
  the server's own interface, and elastic-IP attachments without a data-source lookup.
- Every resource can be imported; resources that live under a parent use composite ids such as
  `server_id:id` (the format is in each resource's documentation). Key pairs and certificates carry
  attributes the api never returns (`public_key`, `certificate`, `private_key`), so the plan after
  their import wants a replace — and a replaced key pair rebuilds the servers referencing it; add
  `lifecycle { ignore_changes = [public_key] }` (or `certificate`, `private_key`) to adopt them.

## v1.1.2 - 2026-08-17
- Fixed `terraform import` for every importable resource: all ids are numeric, but the import wrote
  them as strings and failed schema validation.
- Fixed the docs generation CI (tfplugindocs 0.13 → 0.25 — an expired GPG key broke every docs job),
  updated terraform-plugin-log to 0.11 and the GitHub Actions; the acceptance job is skipped when no
  token is configured so pull requests stop failing.

## v1.1.1 - 2025-11-11
- Require Go 1.25 for local builds.
- Verified dependency stack remains on legacy-compatible Terraform plugin libraries (framework v0.10.0, SDK v2.20.0).
- Added automated `govulncheck ./...` security scan to release pipeline.



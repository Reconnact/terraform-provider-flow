---
page_title: "Known limitations"
subcategory: ""
description: |-
  what the provider cannot do today, and what to do instead
---

# Known limitations

Most of these come from the api, not from the provider.

## Imports and secrets

| What | What to do |
|---|---|
| importing a `flow_compute_key_pair` plans a replace of it and of every `flow_compute_server` that uses it | `lifecycle { ignore_changes = [public_key] }` on the `flow_compute_key_pair` |
| a change to a write-only argument plans nothing: `certificate` and `private_key` on `flow_compute_certificate`, `password` and `cloud_init` on `flow_compute_server`, `password` on `flow_mac_bare_metal_device` | `terraform apply -replace=<address>` |
| `kube_config` of the `flow_kubernetes_kube_config` data source is stored in the state | keep the state in an encrypted backend |

## Kubernetes

| What | What to do |
|---|---|
| `version_id` on `flow_kubernetes_cluster` cannot be set at create | create the `flow_kubernetes_cluster` first, set `version_id` in a later apply |
| the configuration variables of a `flow_kubernetes_cluster` are not managed. a change of `version_id` sends the current variables along, and the apply fails when they do not fit the target version's schema | adjust the variables in the portal first, or change the version there, then apply again |
| the nodes, volumes and load balancers of a `flow_kubernetes_cluster` are read-only: `flow_kubernetes_node`, `flow_kubernetes_volume` and `flow_kubernetes_load_balancer` are data sources | manage them inside the cluster or in the portal |

## Mac bare metal

| What | What to do |
|---|---|
| destroying a `flow_mac_bare_metal_device` only ends its commitment. Terraform reports the destroy as done, but the device keeps running and billing until the commitment period ends, its network interface stays, and the `flow_mac_bare_metal_network` cannot be destroyed before that. the call is made once and the api's answer is final, it is not retried like other deletes | destroy the device first, the network once the period is over |
| an elastic ip attached to a `flow_mac_bare_metal_device` is deleted with the device at the end of its commitment period | detach it with `flow_mac_bare_metal_elastic_ip_device_attachment` before the device is destroyed, Terraform does that on its own when the attachment is a resource |

## Compute

| What | What to do |
|---|---|
| shrinking a `flow_compute_volume` replaces it. the data is lost | back up the data first, or set `lifecycle { prevent_destroy = true }` |
| removing `certificate_id` from a `flow_compute_load_balancer_pool` fails the apply | `terraform apply -replace=<address>` on the `flow_compute_load_balancer_pool` |

## Provider

| What | What to do |
|---|---|
| a request the api rejects only fails after about 90 seconds | `retry_timeout = "0"` or `FLOW_RETRY_TIMEOUT=0` |
| Terraform below 1.11 cannot set the write-only arguments | upgrade Terraform |

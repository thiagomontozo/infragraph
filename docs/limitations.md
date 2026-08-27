# Limitations

- Version 1.0.0-rc.1 is a production candidate until every release gate and remote workflow passes.
- V1 is read-only. It cannot start/stop/delete infrastructure or run arbitrary commands/scripts.
- AWS, Azure, GCP, VMware, Proxmox, Hyper-V, NetBox, ServiceNow, and Ansible adapters are not implemented.
- Rate limiting and SSE fanout are process-local; controlled single-instance deployment is the supported production-candidate topology.
- Docker discovery is the only connector wired into the shipped collector executable. The Kubernetes connector library is tested but not yet selectable by the executable.
- The web application is currently a read/preview operational surface. User/MFA lifecycle, connector/policy administration, import apply, export generation, and merge review still require complete workflows before the broad product scope can be promoted to 1.0.
- Merge history is preserved, but universal automatic merge reversal is not implemented because dependent state may have changed.
- Docker socket possession remains host-equivalent risk; read-only mounts are not a sufficient boundary.
- Terraform raw state is not retained by default, so later forensic re-parsing requires re-import from the original protected source.
- Performance smoke is bounded and makes no universal capacity/SLA claim.


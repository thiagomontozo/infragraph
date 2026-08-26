# Connectors

V1 connector types are Manual, CSV, JSON Import, Docker, Kubernetes, Terraform State, and NetScope Import. Discovery connectors advertise granular capabilities and are read-only. Configuration is validated into a typed connector-specific structure; no public API accepts a raw infrastructure endpoint path, arbitrary HTTP verb, shell command, or script.

Future adapters may target AWS, Azure, GCP, VMware, Proxmox, Hyper-V, NetBox, ServiceNow, and Ansible behind the same normalization interface. They are roadmap items, not simulated integrations.


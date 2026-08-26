# Terraform state import

Terraform state often contains credentials and sensitive application material. InfraGraph does not persist raw state by default. The streaming size boundary is applied before parsing, outputs are ignored, and resource extraction uses an allowlist: address/type/provider plus identifiers, name/ARN, region/location/zone, namespace/cluster, and tags/labels.

Keys containing password, secret, token, private key, connection string, user data, or credential are excluded even if a future allowlist accidentally names them. Arbitrary resource attributes, sensitive outputs, provisioner connection details, and raw state artifacts are not stored. Preview shows only the sanitized observations before confirmation.


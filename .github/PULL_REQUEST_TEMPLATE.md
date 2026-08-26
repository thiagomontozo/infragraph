## Summary and risk

Describe user-visible behavior, trust-boundary/data-collection changes, tenant impact, migrations, and backward compatibility.

## Checklist

- [ ] Tests cover the behavior and critical invariants remain enabled
- [ ] Documentation/OpenAPI/contracts are updated
- [ ] Security and threat-model impact was reviewed
- [ ] Migrations are versioned, locked, tested, and rollback/forward repair is documented
- [ ] No secrets, real snapshots, or production Terraform state are included
- [ ] Organization predicates and permission checks are present
- [ ] Collector changes remain read-only and minimize data
- [ ] Backward compatibility and release notes were considered


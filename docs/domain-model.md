# Domain model

An organization owns users, collectors, connectors, assets, source identities, observations, claims, relationships, snapshots, policies, changes, findings, artifacts, and audit events. Browser-provided organization IDs are ignored; session and collector credentials determine the boundary.

```mermaid
erDiagram
  ORGANIZATION ||--o{ USER : owns
  ORGANIZATION ||--o{ ASSET : contains
  ASSET ||--o{ ASSET_SOURCE_IDENTITY : correlates
  CONNECTOR ||--o{ ASSET_SOURCE_IDENTITY : reports
  CONNECTOR ||--o{ SOURCE_SNAPSHOT : produces
  SOURCE_SNAPSHOT ||--o{ ASSET_OBSERVATION : contains
  ASSET ||--o{ ATTRIBUTE_CLAIM : has
  ASSET ||--o{ RELATIONSHIP : from
  ASSET ||--o{ RELATIONSHIP : to
  SOURCE_SNAPSHOT ||--|| RECONCILIATION_RUN : drives
  ASSET ||--o{ INFRASTRUCTURE_CHANGE : changes
  ASSET ||--o{ INFRASTRUCTURE_FINDING : raises
```

Snapshots are immutable evidence metadata. Reconciliation updates canonical state atomically only after validation. Claims and observations retain provenance; a merge moves references but never erases evidence. Findings explain data-quality conditions and remain distinct from infrastructure compliance claims.


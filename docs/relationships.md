# Relationships

Relationships connect two assets within one organization and retain first/last-seen times and status. Initial types cover hosting, dependencies, connections, databases, caches, storage, images, volumes, DNS, routing, membership, management, backing, and parent/child structure. Relationship observations retain connector and snapshot provenance.

An edge means the source observed or declared a relationship. It does not prove causation or guaranteed outage. UI and API language therefore says “depends on”, “observed dependency”, and “potentially affected”. Removal requires comparison between successful relevant snapshots; a failed run leaves the previous edge inconclusive.


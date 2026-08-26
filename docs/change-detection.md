# Change detection

Changes include asset discovery/missing/return/retirement, attribute and ownership/environment changes, relationship add/remove, source conflict/resolution, and stale connectors. Each event stores before/after JSON, source, detection time, summary, confidence, and a stable logical identity.

Detection compares the newly committed effective state with the preceding effective state inside the reconciliation transaction. Retrying the same snapshot hits the logical identity constraint instead of duplicating history. Timeline filters operate by asset, source, type, environment, and time.


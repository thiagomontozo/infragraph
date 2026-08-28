# Frontend

React, TypeScript, Vite, React Router, TanStack Query, and `@xyflow/react` implement the operational UI. Pages expose loading, error, and empty states; an error boundary prevents a blank page and routes failures through an abstraction that can later integrate Sentry without making it mandatory.

The version 1.0 Asset 360 surface exposes effective-state overview plus bounded relationship/dependency views. Navigation is deliberately limited to API-backed read-only pages; there are no placeholder administrative routes or sample claims presented as real data. Graph rendering is query-driven and bounded, with depth controls and a semantic relationship table. Import pages validate and preview but never apply. See the [version 1.0 product scope](product-scope.md).

Keyboard focus, form labels, table headers, contrast, reduced motion, and tablet layouts are built in. Session material stays in Secure/HttpOnly cookies, never localStorage; React escaping is retained and no unsafe HTML renderer is used.


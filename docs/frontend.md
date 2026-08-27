# Frontend

React, TypeScript, Vite, React Router, TanStack Query, and `@xyflow/react` implement the operational UI. Pages expose loading, error, and empty states; an error boundary prevents a blank page and routes failures through an abstraction that can later integrate Sentry without making it mandatory.

The current Asset 360 implementation exposes overview plus bounded relationship/dependency views; the broader provenance, evidence, timeline, and administrative workflows remain release-candidate work. Graph rendering is query-driven and bounded, with depth controls and a semantic relationship table. Keyboard focus, form labels, table headers, contrast, reduced motion, and tablet layouts are built in. Session material stays in Secure/HttpOnly cookies, never localStorage; React escaping is retained and no unsafe HTML renderer is used.


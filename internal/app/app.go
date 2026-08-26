package app

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/infragraph/internal/config"
	"github.com/thiagomontozo/infragraph/internal/database"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"github.com/thiagomontozo/infragraph/internal/graph"
	"github.com/thiagomontozo/infragraph/internal/imports"
	"github.com/thiagomontozo/infragraph/internal/security"
)

type contextKey string

const principalKey contextKey = "principal"

type principal struct {
	UserID, OrganizationID string
	Permissions            map[string]bool
	CSRFHash               string
}
type metrics struct {
	requests          atomic.Uint64
	failures          atomic.Uint64
	signatureFailures atomic.Uint64
	graphRejections   atomic.Uint64
}
type limiter struct {
	mu      sync.Mutex
	windows map[string]window
	limit   int
	period  time.Duration
}
type window struct {
	start time.Time
	count int
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	w := l.windows[key]
	if now.Sub(w.start) > l.period {
		w = window{start: now}
	}
	w.count++
	l.windows[key] = w
	return w.count <= l.limit
}

type App struct {
	cfg                          config.Config
	db                           *database.DB
	log                          *slog.Logger
	started                      time.Time
	metrics                      metrics
	authLimiter, snapshotLimiter *limiter
	graph                        *graph.Service
}

func New(cfg config.Config, db *database.DB, logger *slog.Logger) *App {
	if logger == nil {
		logger = slog.Default()
	}
	return &App{cfg: cfg, db: db, log: logger, started: time.Now(), authLimiter: &limiter{windows: map[string]window{}, limit: 10, period: time.Minute}, snapshotLimiter: &limiter{windows: map[string]window{}, limit: 30, period: time.Minute}, graph: graph.New(graph.Limits{MaxDepth: cfg.MaxGraphDepth, MaxNodes: cfg.MaxGraphNodes, Timeout: 2 * time.Second})}
}
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /ready", a.ready)
	mux.HandleFunc("GET /startup", a.startup)
	mux.HandleFunc("GET /metrics", a.prometheus)
	mux.HandleFunc("POST /api/v1/auth/login", a.login)
	mux.HandleFunc("POST /collector/v1/enroll", a.enroll)
	mux.HandleFunc("POST /collector/v1/heartbeat", a.collectorHeartbeat)
	mux.HandleFunc("POST /collector/v1/snapshots", a.collectorSnapshot)
	mux.Handle("/api/v1/", a.session(http.HandlerFunc(a.api)))
	return a.middleware(mux)
}

func (a *App) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = newID("req")
		}
		r.Header.Set("X-Request-ID", rid)
		w.Header().Set("X-Request-ID", rid)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if a.cfg.Environment == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		origin := r.Header.Get("Origin")
		if origin != "" && contains(a.cfg.AllowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if origin == "" || !contains(a.cfg.AllowedOrigins, origin) {
				writeError(w, http.StatusForbidden, "origin_denied", "origin is not allowed", rid)
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-CSRF-Token,X-Request-ID,Idempotency-Key")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		a.metrics.requests.Add(1)
		next.ServeHTTP(w, r)
		a.log.Info("http_request", "requestId", rid, "method", r.Method, "path", r.URL.Path, "durationMs", time.Since(start).Milliseconds())
	})
}
func (a *App) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok"})
}
func (a *App) startup(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "started", "startedAt": a.started})
}
func (a *App) ready(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, 503, "database_unavailable", "database is not configured", requestID(r))
		return
	}
	ctx, c := context.WithTimeout(r.Context(), 2*time.Second)
	defer c()
	if e := a.db.Ready(ctx); e != nil {
		writeError(w, 503, "not_ready", "essential dependency unavailable", requestID(r))
		return
	}
	writeJSON(w, 200, map[string]any{"status": "ready"})
}
func (a *App) prometheus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# TYPE http_requests_total counter\nhttp_requests_total %d\n# TYPE http_failures_total counter\nhttp_failures_total %d\n# TYPE collector_signature_failures_total counter\ncollector_signature_failures_total %d\n# TYPE graph_query_limit_rejections_total counter\ngraph_query_limit_rejections_total %d\n", a.metrics.requests.Load(), a.metrics.failures.Load(), a.metrics.signatureFailures.Load(), a.metrics.graphRejections.Load())
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if !a.authLimiter.allow(r.RemoteAddr) {
		writeError(w, 429, "rate_limited", "too many authentication attempts", requestID(r))
		return
	}
	if a.db == nil {
		writeError(w, 503, "database_unavailable", "authentication unavailable", requestID(r))
		return
	}
	var in struct{ Email, Password, TOTP, RecoveryCode string }
	if e := decode(w, r, &in, 64<<10); e != nil {
		return
	}
	var userID, orgID, hash string
	var active, mfaRequired bool
	e := a.db.Pool.QueryRow(r.Context(), "SELECT id,organization_id,password_hash,active,mfa_required FROM users WHERE lower(email)=lower($1)", in.Email).Scan(&userID, &orgID, &hash, &active, &mfaRequired)
	if e != nil || !active || !security.VerifyPassword(hash, in.Password) {
		time.Sleep(75 * time.Millisecond)
		writeError(w, 401, "invalid_credentials", "invalid credentials", requestID(r))
		return
	}
	if mfaRequired {
		if !a.verifySecondFactor(r.Context(), userID, in.TOTP, in.RecoveryCode) {
			writeError(w, 401, "mfa_required", "a valid second factor is required", requestID(r))
			return
		}
	}
	token, _ := security.RandomToken(32)
	csrf, _ := security.RandomToken(32)
	expires := time.Now().Add(12 * time.Hour)
	_, e = a.db.Pool.Exec(r.Context(), "INSERT INTO sessions(id_hash,user_id,organization_id,csrf_hash,expires_at) VALUES($1,$2,$3,$4,$5)", security.TokenHash(token), userID, orgID, security.TokenHash(csrf), expires)
	if e != nil {
		writeError(w, 500, "internal_error", "could not create session", requestID(r))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "infragraph_session", Value: token, Path: "/", HttpOnly: true, Secure: a.cfg.Environment == "production", SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: 43200})
	http.SetCookie(w, &http.Cookie{Name: "infragraph_csrf", Value: csrf, Path: "/", HttpOnly: false, Secure: a.cfg.Environment == "production", SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: 43200})
	writeJSON(w, 200, map[string]any{"user": map[string]string{"id": userID}, "expiresAt": expires})
}

func (a *App) verifySecondFactor(ctx context.Context, userID, code, recovery string) bool {
	if recovery != "" {
		command, err := a.db.Pool.Exec(ctx, "UPDATE recovery_codes SET used_at=now() WHERE user_id=$1 AND code_hash=$2 AND used_at IS NULL", userID, security.TokenHash(recovery))
		if err == nil && command.RowsAffected() == 1 {
			return true
		}
	}
	var ciphertext, nonce string
	var version int
	if err := a.db.Pool.QueryRow(ctx, "SELECT ciphertext,nonce,key_version FROM mfa_secrets WHERE user_id=$1 AND enabled_at IS NOT NULL", userID).Scan(&ciphertext, &nonce, &version); err != nil {
		return false
	}
	key, err := base64.StdEncoding.DecodeString(a.cfg.MasterKey)
	if err != nil {
		return false
	}
	store, err := security.NewSecretStore(key, version)
	if err != nil {
		return false
	}
	secret, err := store.Decrypt(security.SecretEnvelope{Ciphertext: ciphertext, Nonce: nonce, KeyVersion: version})
	if err != nil {
		return false
	}
	return security.VerifyTOTP(string(secret), code, time.Now())
}

func (a *App) session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.db == nil {
			writeError(w, 503, "database_unavailable", "database unavailable", requestID(r))
			return
		}
		cookie, e := r.Cookie("infragraph_session")
		if e != nil {
			writeError(w, 401, "unauthenticated", "authentication required", requestID(r))
			return
		}
		p := principal{Permissions: map[string]bool{}}
		e = a.db.Pool.QueryRow(r.Context(), "SELECT s.user_id,s.organization_id,s.csrf_hash FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.active", security.TokenHash(cookie.Value)).Scan(&p.UserID, &p.OrganizationID, &p.CSRFHash)
		if e != nil {
			writeError(w, 401, "unauthenticated", "session invalid or expired", requestID(r))
			return
		}
		rows, e := a.db.Pool.Query(r.Context(), "SELECT p.name FROM user_roles ur JOIN role_permissions rp ON rp.role_id=ur.role_id JOIN permissions p ON p.id=rp.permission_id WHERE ur.user_id=$1", p.UserID)
		if e == nil {
			defer rows.Close()
			for rows.Next() {
				var v string
				if rows.Scan(&v) == nil {
					p.Permissions[v] = true
				}
			}
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			csrf := r.Header.Get("X-CSRF-Token")
			if subtle.ConstantTimeCompare([]byte(security.TokenHash(csrf)), []byte(p.CSRFHash)) != 1 {
				writeError(w, 403, "csrf_invalid", "CSRF token is missing or invalid", requestID(r))
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	})
}
func principalFrom(r *http.Request) principal { return r.Context().Value(principalKey).(principal) }
func require(w http.ResponseWriter, r *http.Request, p string) bool {
	if !principalFrom(r).Permissions[p] {
		writeError(w, 403, "permission_denied", "required permission is missing", requestID(r))
		return false
	}
	return true
}

func (a *App) api(w http.ResponseWriter, r *http.Request) {
	route := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	switch {
	case r.Method == "GET" && route == "auth/me":
		p := principalFrom(r)
		writeJSON(w, 200, map[string]any{"id": p.UserID, "organizationId": p.OrganizationID, "permissions": p.Permissions})
	case r.Method == "POST" && route == "auth/logout":
		p := principalFrom(r)
		c, _ := r.Cookie("infragraph_session")
		a.db.Pool.Exec(r.Context(), "UPDATE sessions SET revoked_at=now() WHERE id_hash=$1 AND organization_id=$2", security.TokenHash(c.Value), p.OrganizationID)
		http.SetCookie(w, &http.Cookie{Name: "infragraph_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
		w.WriteHeader(204)
	case r.Method == "GET" && route == "asset-types":
		writeJSON(w, 200, domain.AssetTypeLabels)
	case r.Method == "GET" && route == "overview":
		a.overview(w, r)
	case r.Method == "GET" && route == "assets":
		a.assets(w, r)
	case r.Method == "GET" && strings.HasPrefix(route, "assets/"):
		a.assetRoute(w, r, route)
	case r.Method == "POST" && route == "imports/csv/preview":
		a.csvPreview(w, r)
	case r.Method == "POST" && route == "imports/json/preview":
		a.jsonPreview(w, r)
	case r.Method == "POST" && route == "imports/terraform/preview":
		a.terraformPreview(w, r)
	case r.Method == "GET" && route == "changes":
		a.genericList(w, r, "infrastructure_changes", "change_type,summary,detected_at", "detected_at")
	case r.Method == "GET" && route == "findings":
		a.genericList(w, r, "infrastructure_findings", "finding_type,status,priority,explanation,created_at", "created_at")
	case r.Method == "GET" && route == "connectors":
		a.genericList(w, r, "infrastructure_connectors", "name,type,enabled,authoritative_level,last_status,last_successful_sync_at", "created_at")
	case r.Method == "GET" && route == "collectors":
		a.genericList(w, r, "collectors", "name,status,collector_version,protocol_version,compatibility_status,last_heartbeat_at,fingerprint", "enrolled_at")
	case r.Method == "GET" && route == "audit":
		if require(w, r, "audit.read") {
			a.genericList(w, r, "audit_events", "action,resource_type,resource_id,request_id,occurred_at,event_hash", "occurred_at")
		}
	default:
		writeError(w, 404, "not_found", "endpoint not found", requestID(r))
	}
}

func (a *App) overview(w http.ResponseWriter, r *http.Request) {
	if !require(w, r, "asset.read") {
		return
	}
	org := principalFrom(r).OrganizationID
	var assets, relationships, active, stale, missing, conflicts int
	e := a.db.Pool.QueryRow(r.Context(), "SELECT count(*),count(*) FILTER(WHERE status='ACTIVE'),count(*) FILTER(WHERE status='STALE'),count(*) FILTER(WHERE status='MISSING'),count(*) FILTER(WHERE status='CONFLICTING') FROM assets WHERE organization_id=$1", org).Scan(&assets, &active, &stale, &missing, &conflicts)
	if e != nil {
		writeError(w, 500, "query_failed", "could not load overview", requestID(r))
		return
	}
	a.db.Pool.QueryRow(r.Context(), "SELECT count(*) FROM asset_relationships WHERE organization_id=$1 AND status='ACTIVE'", org).Scan(&relationships)
	writeJSON(w, 200, map[string]int{"assets": assets, "relationships": relationships, "active": active, "stale": stale, "missing": missing, "conflicts": conflicts})
}
func (a *App) assets(w http.ResponseWriter, r *http.Request) {
	if !require(w, r, "asset.read") {
		return
	}
	p := principalFrom(r)
	limit := boundedInt(r.URL.Query().Get("limit"), 50, 1, 200)
	offset := boundedInt(r.URL.Query().Get("offset"), 0, 0, 100000)
	q := "%" + r.URL.Query().Get("q") + "%"
	rows, e := a.db.Pool.Query(r.Context(), "SELECT id,canonical_name,display_name,asset_type,status,coalesce(environment,''),criticality,first_seen_at,last_seen_at FROM assets WHERE organization_id=$1 AND ($2='%%' OR canonical_name ILIKE $2 OR display_name ILIKE $2) ORDER BY canonical_name LIMIT $3 OFFSET $4", p.OrganizationID, q, limit, offset)
	if e != nil {
		writeError(w, 500, "query_failed", "could not list assets", requestID(r))
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, canonical, display, typ, status, env, criticality string
		var first, last time.Time
		if rows.Scan(&id, &canonical, &display, &typ, &status, &env, &criticality, &first, &last) == nil {
			items = append(items, map[string]any{"id": id, "canonicalName": canonical, "displayName": display, "assetType": typ, "status": status, "environment": env, "criticality": criticality, "firstSeenAt": first, "lastSeenAt": last})
		}
	}
	writeJSON(w, 200, map[string]any{"items": items, "limit": limit, "offset": offset})
}
func (a *App) assetRoute(w http.ResponseWriter, r *http.Request, route string) {
	if !require(w, r, "asset.read") {
		return
	}
	parts := strings.Split(route, "/")
	id := parts[1]
	org := principalFrom(r).OrganizationID
	if len(parts) == 2 {
		var v map[string]any = map[string]any{}
		var name, display, typ, status, env, criticality string
		var first, last time.Time
		e := a.db.Pool.QueryRow(r.Context(), "SELECT canonical_name,display_name,asset_type,status,coalesce(environment,''),criticality,first_seen_at,last_seen_at FROM assets WHERE id=$1 AND organization_id=$2", id, org).Scan(&name, &display, &typ, &status, &env, &criticality, &first, &last)
		if e == pgx.ErrNoRows {
			writeError(w, 404, "not_found", "asset not found", requestID(r))
			return
		}
		if e != nil {
			writeError(w, 500, "query_failed", "could not load asset", requestID(r))
			return
		}
		v = map[string]any{"id": id, "canonicalName": name, "displayName": display, "assetType": typ, "status": status, "environment": env, "criticality": criticality, "firstSeenAt": first, "lastSeenAt": last}
		writeJSON(w, 200, v)
		return
	}
	kind := parts[2]
	reverse := kind == "dependents" || kind == "impact"
	depth := boundedInt(r.URL.Query().Get("maxDepth"), 3, 1, a.cfg.MaxGraphDepth)
	nodes := boundedInt(r.URL.Query().Get("maxNodes"), 100, 1, a.cfg.MaxGraphNodes)
	rows, e := a.db.Pool.Query(r.Context(), "SELECT id,from_asset_id,to_asset_id,type,status,first_seen_at,last_seen_at,created_at,updated_at FROM asset_relationships WHERE organization_id=$1", org)
	if e != nil {
		writeError(w, 500, "query_failed", "could not load graph", requestID(r))
		return
	}
	defer rows.Close()
	var edges []domain.Relationship
	for rows.Next() {
		var edge domain.Relationship
		edge.OrganizationID = org
		if rows.Scan(&edge.ID, &edge.FromAssetID, &edge.ToAssetID, &edge.Type, &edge.Status, &edge.FirstSeenAt, &edge.LastSeenAt, &edge.CreatedAt, &edge.UpdatedAt) == nil {
			edges = append(edges, edge)
		}
	}
	result, e := a.graph.Traverse(r.Context(), org, id, depth, nodes, reverse, edges)
	if e != nil {
		if errors.Is(e, graph.ErrLimitExceeded) {
			a.metrics.graphRejections.Add(1)
			writeError(w, 422, "graph_limit_exceeded", e.Error(), requestID(r))
		} else {
			writeError(w, 500, "graph_failed", "graph query failed", requestID(r))
		}
		return
	}
	writeJSON(w, 200, result)
}

func (a *App) genericList(w http.ResponseWriter, r *http.Request, table, columns, order string) {
	org := principalFrom(r).OrganizationID
	limit := boundedInt(r.URL.Query().Get("limit"), 50, 1, 200)
	query := "SELECT row_to_json(x) FROM (SELECT id," + columns + " FROM " + table + " WHERE organization_id=$1 ORDER BY " + order + " DESC LIMIT $2) x"
	rows, e := a.db.Pool.Query(r.Context(), query, org, limit)
	if e != nil {
		writeError(w, 500, "query_failed", "could not load records", requestID(r))
		return
	}
	defer rows.Close()
	items := []json.RawMessage{}
	for rows.Next() {
		var v json.RawMessage
		if rows.Scan(&v) == nil {
			items = append(items, v)
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (a *App) csvPreview(w http.ResponseWriter, r *http.Request) {
	if !require(w, r, "import.run") {
		return
	}
	p, e := imports.CSV(r.Body, imports.Limits{MaxBytes: a.cfg.MaxImportBytes, MaxRows: 10000})
	if e != nil {
		writeError(w, 422, "invalid_csv", e.Error(), requestID(r))
		return
	}
	writeJSON(w, 200, p)
}
func (a *App) jsonPreview(w http.ResponseWriter, r *http.Request) {
	if !require(w, r, "import.run") {
		return
	}
	p, e := imports.JSON(r.Body, imports.Limits{MaxBytes: a.cfg.MaxImportBytes, MaxItems: 10000, MaxDepth: 16})
	if e != nil {
		writeError(w, 422, "invalid_json", e.Error(), requestID(r))
		return
	}
	writeJSON(w, 200, p)
}
func (a *App) terraformPreview(w http.ResponseWriter, r *http.Request) {
	if !require(w, r, "import.run") {
		return
	}
	p, e := imports.Terraform(r.Body, a.cfg.MaxImportBytes)
	if e != nil {
		writeError(w, 422, "invalid_terraform_state", e.Error(), requestID(r))
		return
	}
	writeJSON(w, 200, p)
}

func (a *App) enroll(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, 503, "database_unavailable", "enrollment unavailable", requestID(r))
		return
	}
	if !a.authLimiter.allow(r.RemoteAddr) {
		writeError(w, 429, "rate_limited", "too many enrollment attempts", requestID(r))
		return
	}
	var in struct{ Token, Name, PublicKey, CollectorVersion, ProtocolVersion string }
	if e := decode(w, r, &in, 64<<10); e != nil {
		return
	}
	raw, e := base64.StdEncoding.DecodeString(in.PublicKey)
	if e != nil || len(raw) != ed25519.PublicKeySize {
		writeError(w, 422, "invalid_public_key", "Ed25519 public key required", requestID(r))
		return
	}
	tx, e := a.db.Pool.Begin(r.Context())
	if e != nil {
		writeError(w, 500, "internal_error", "enrollment failed", requestID(r))
		return
	}
	defer tx.Rollback(r.Context())
	var org string
	e = tx.QueryRow(r.Context(), "UPDATE collector_enrollment_tokens SET used_at=now() WHERE token_hash=$1 AND used_at IS NULL AND expires_at>now() AND scope='collector.enroll' RETURNING organization_id", security.TokenHash(in.Token)).Scan(&org)
	if e != nil {
		writeError(w, 401, "invalid_enrollment_token", "token invalid, expired, or already used", requestID(r))
		return
	}
	finger := sha256.Sum256(raw)
	collectorID := newID("collector")
	credential, _ := security.RandomToken(32)
	compat := "COMPATIBLE"
	if !strings.HasPrefix(in.ProtocolVersion, "1.") {
		compat = "INCOMPATIBLE"
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO collectors(id,organization_id,name,public_key,fingerprint,status,collector_version,protocol_version,compatibility_status) VALUES($1,$2,$3,$4,$5,'ACTIVE',$6,$7,$8)", collectorID, org, in.Name, raw, hex.EncodeToString(finger[:]), in.CollectorVersion, in.ProtocolVersion, compat)
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO collector_credentials(id,organization_id,collector_id,credential_hash) VALUES($1,$2,$3,$4)", newID("credential"), org, collectorID, security.TokenHash(credential))
	}
	if e != nil || tx.Commit(r.Context()) != nil {
		writeError(w, 500, "internal_error", "enrollment failed", requestID(r))
		return
	}
	writeJSON(w, 201, map[string]any{"collectorId": collectorID, "organizationId": org, "credential": credential, "compatibilityStatus": compat})
}
func (a *App) collectorPrincipal(r *http.Request) (string, string, ed25519.PublicKey, bool) {
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if auth == "" || a.db == nil {
		return "", "", nil, false
	}
	var id, org string
	var pub []byte
	var revoked *time.Time
	e := a.db.Pool.QueryRow(r.Context(), "SELECT c.id,c.organization_id,c.public_key,c.revoked_at FROM collector_credentials cc JOIN collectors c ON c.id=cc.collector_id WHERE cc.credential_hash=$1 AND cc.revoked_at IS NULL AND (cc.expires_at IS NULL OR cc.expires_at>now())", security.TokenHash(auth)).Scan(&id, &org, &pub, &revoked)
	return id, org, ed25519.PublicKey(pub), e == nil && revoked == nil
}
func (a *App) collectorHeartbeat(w http.ResponseWriter, r *http.Request) {
	id, org, _, ok := a.collectorPrincipal(r)
	if !ok {
		writeError(w, 401, "collector_unauthorized", "collector credential invalid or revoked", requestID(r))
		return
	}
	var in struct {
		CollectorVersion, ProtocolVersion, OS, Architecture string
		Capabilities                                        []string
		RunningJobs                                         int
		HealthSummary                                       string
	}
	if e := decode(w, r, &in, 128<<10); e != nil {
		return
	}
	compat := "COMPATIBLE"
	if !strings.HasPrefix(in.ProtocolVersion, "1.") {
		compat = "INCOMPATIBLE"
	}
	_, e := a.db.Pool.Exec(r.Context(), "UPDATE collectors SET collector_version=$1,protocol_version=$2,os=$3,architecture=$4,capabilities=$5,compatibility_status=$6,last_heartbeat_at=now() WHERE id=$7 AND organization_id=$8", in.CollectorVersion, in.ProtocolVersion, in.OS, in.Architecture, in.Capabilities, compat, id, org)
	if e != nil {
		writeError(w, 500, "internal_error", "heartbeat update failed", requestID(r))
		return
	}
	writeJSON(w, 200, map[string]any{"compatibilityStatus": compat, "serverTime": time.Now().UTC()})
}
func (a *App) collectorSnapshot(w http.ResponseWriter, r *http.Request) {
	id, org, pub, ok := a.collectorPrincipal(r)
	if !ok {
		writeError(w, 401, "collector_unauthorized", "collector credential invalid or revoked", requestID(r))
		return
	}
	if !a.snapshotLimiter.allow(id) {
		writeError(w, 429, "rate_limited", "snapshot ingest rate exceeded", requestID(r))
		return
	}
	var s domain.SnapshotEnvelope
	if e := decode(w, r, &s, a.cfg.MaxSnapshotBytes); e != nil {
		return
	}
	if s.CollectorID != id || s.OrganizationID != org {
		writeError(w, 403, "collector_binding_mismatch", "snapshot identity does not match credential", requestID(r))
		return
	}
	if time.Since(s.CompletedAt) > 24*time.Hour || s.CompletedAt.After(time.Now().Add(5*time.Minute)) {
		writeError(w, 409, "snapshot_timestamp_invalid", "snapshot timestamp outside replay window", requestID(r))
		return
	}
	if e := security.VerifySnapshot(s, pub); e != nil {
		a.metrics.signatureFailures.Add(1)
		writeError(w, 401, "invalid_snapshot_signature", e.Error(), requestID(r))
		return
	}
	var maxSequence int64
	a.db.Pool.QueryRow(r.Context(), "SELECT coalesce(max(sequence),0) FROM source_snapshots WHERE organization_id=$1 AND collector_id=$2", org, id).Scan(&maxSequence)
	if s.Sequence <= maxSequence {
		writeError(w, 409, "snapshot_replay", "snapshot sequence was already observed", requestID(r))
		return
	}
	_, e := a.db.Pool.Exec(r.Context(), "INSERT INTO source_snapshots(id,organization_id,connector_id,collector_id,status,sequence,started_at,completed_at,asset_count,relationship_count,content_hash,protocol_version) VALUES($1,$2,$3,$4,'QUEUED',$5,$6,$7,$8,$9,$10,$11) ON CONFLICT DO NOTHING", s.SnapshotID, org, s.ConnectorID, id, s.Sequence, s.StartedAt, s.CompletedAt, len(s.Assets), len(s.Relationships), s.ContentHash, s.ProtocolVersion)
	if e != nil {
		writeError(w, 422, "snapshot_rejected", "snapshot could not be staged", requestID(r))
		return
	}
	writeJSON(w, 202, map[string]any{"snapshotId": s.SnapshotID, "status": "QUEUED"})
}

func decode(w http.ResponseWriter, r *http.Request, out any, max int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(out); e != nil {
		writeError(w, 400, "invalid_request", "request body is invalid", requestID(r))
		return e
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, message, rid string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": rid}})
}
func requestID(r *http.Request) string { return r.Header.Get("X-Request-ID") }
func newID(prefix string) string {
	b := make([]byte, 16)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
func boundedInt(v string, d, min, max int) int {
	n, e := strconv.Atoi(v)
	if e != nil {
		return d
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

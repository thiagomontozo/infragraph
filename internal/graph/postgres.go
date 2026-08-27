package graph

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thiagomontozo/infragraph/internal/domain"
)

// TraversePostgres performs an indexed, bounded breadth-first traversal so API
// memory and database-to-API traffic do not grow with the tenant's full graph.
func TraversePostgres(ctx context.Context, pool *pgxpool.Pool, organizationID, start string, depth, maxNodes int, reverse bool) (Result, error) {
	if pool == nil || organizationID == "" || start == "" {
		return Result{}, errors.New("database, organization, and start asset are required")
	}
	if depth < 1 || maxNodes < 1 {
		return Result{}, ErrLimitExceeded
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND organization_id=$2)`, start, organizationID).Scan(&exists); err != nil {
		return Result{}, err
	}
	if !exists {
		return Result{}, pgx.ErrNoRows
	}
	result := Result{Nodes: []string{start}}
	seen := map[string]bool{start: true}
	frontier := []string{start}
	for level := 1; level <= depth && len(frontier) > 0; level++ {
		visited := append([]string(nil), result.Nodes...)
		remaining := maxNodes - len(result.Nodes)
		rows, queryErr := pool.Query(ctx, `SELECT DISTINCT CASE WHEN $4::boolean THEN from_asset_id ELSE to_asset_id END AS target FROM asset_relationships WHERE organization_id=$1 AND status='ACTIVE' AND (CASE WHEN $4::boolean THEN to_asset_id ELSE from_asset_id END)=ANY($2) AND NOT (CASE WHEN $4::boolean THEN from_asset_id ELSE to_asset_id END)=ANY($3) ORDER BY target LIMIT $5`, organizationID, frontier, visited, reverse, remaining+1)
		if queryErr != nil {
			return Result{}, queryErr
		}
		next := []string{}
		for rows.Next() {
			var node string
			if scanErr := rows.Scan(&node); scanErr != nil {
				rows.Close()
				return Result{}, scanErr
			}
			if !seen[node] {
				seen[node] = true
				next = append(next, node)
			}
		}
		rows.Close()
		if rowErr := rows.Err(); rowErr != nil {
			return Result{}, rowErr
		}
		if len(next) > remaining {
			return Result{}, fmt.Errorf("%w: maxNodes=%d", ErrLimitExceeded, maxNodes)
		}
		if len(next) > 0 {
			result.Nodes = append(result.Nodes, next...)
			result.Depth = level
		}
		frontier = next
	}
	edgeRows, err := pool.Query(ctx, `SELECT id,organization_id,from_asset_id,to_asset_id,type,status,first_seen_at,last_seen_at,created_at,updated_at FROM asset_relationships WHERE organization_id=$1 AND status='ACTIVE' AND from_asset_id=ANY($2) AND to_asset_id=ANY($2) ORDER BY id`, organizationID, result.Nodes)
	if err != nil {
		return Result{}, err
	}
	defer edgeRows.Close()
	for edgeRows.Next() {
		var edge domain.Relationship
		if err = edgeRows.Scan(&edge.ID, &edge.OrganizationID, &edge.FromAssetID, &edge.ToAssetID, &edge.Type, &edge.Status, &edge.FirstSeenAt, &edge.LastSeenAt, &edge.CreatedAt, &edge.UpdatedAt); err != nil {
			return Result{}, err
		}
		result.Relationships = append(result.Relationships, edge)
	}
	return result, edgeRows.Err()
}

package graph

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thiagomontozo/infragraph/internal/domain"
)

var ErrLimitExceeded = errors.New("graph traversal limit exceeded")

type Limits struct {
	MaxDepth, MaxNodes int
	Timeout            time.Duration
}
type Result struct {
	Nodes         []string              `json:"nodes"`
	Relationships []domain.Relationship `json:"relationships"`
	Depth         int                   `json:"depth"`
	Truncated     bool                  `json:"truncated"`
}
type Service struct{ limits Limits }

func New(l Limits) *Service {
	if l.Timeout <= 0 {
		l.Timeout = 2 * time.Second
	}
	return &Service{l}
}
func (s *Service) Traverse(ctx context.Context, organizationID, start string, depth, nodes int, reverse bool, edges []domain.Relationship) (Result, error) {
	if organizationID == "" || start == "" {
		return Result{}, errors.New("organization and start asset are required")
	}
	if depth < 1 || depth > s.limits.MaxDepth || nodes < 1 || nodes > s.limits.MaxNodes {
		return Result{}, ErrLimitExceeded
	}
	ctx, cancel := context.WithTimeout(ctx, s.limits.Timeout)
	defer cancel()
	seen := map[string]bool{start: true}
	frontier := []string{start}
	out := Result{Nodes: []string{start}}
	for d := 1; d <= depth && len(frontier) > 0; d++ {
		next := []string{}
		for _, id := range frontier {
			for _, e := range edges {
				if e.OrganizationID != organizationID || e.Status != "ACTIVE" {
					continue
				}
				from, to := e.FromAssetID, e.ToAssetID
				if reverse {
					from, to = to, from
				}
				if from != id {
					continue
				}
				out.Relationships = append(out.Relationships, e)
				if !seen[to] {
					if len(seen) >= nodes {
						return Result{}, fmt.Errorf("%w: maxNodes=%d", ErrLimitExceeded, nodes)
					}
					seen[to] = true
					out.Nodes = append(out.Nodes, to)
					next = append(next, to)
				}
				select {
				case <-ctx.Done():
					return Result{}, ctx.Err()
				default:
				}
			}
		}
		frontier = next
		out.Depth = d
	}
	return out, nil
}

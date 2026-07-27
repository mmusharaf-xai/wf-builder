package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var nodeRefPattern = regexp.MustCompile(`\$\(['"]([^'"]+)['"]\)\.(currentItem|firstItem|lastItem|all)\.json`)

type ParseValuesService struct {
	pool *pgxpool.Pool
}

func NewParseValuesService(pool *pgxpool.Pool) *ParseValuesService {
	return &ParseValuesService{pool: pool}
}

// GetParsedValues replaces $("Node Name").currentItem.json-style references with
// JSON literals from completed node executions (ported from helpers.ts).
func (s *ParseValuesService) GetParsedValues(ctx context.Context, executionID, code string) (string, error) {
	matches := nodeRefPattern.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return code, nil
	}

	nameSet := map[string]struct{}{}
	type ref struct {
		full   string
		name   string
		method string
	}
	refs := make([]ref, 0, len(matches))
	for _, m := range matches {
		refs = append(refs, ref{full: m[0], name: m[1], method: m[2]})
		nameSet[m[1]] = struct{}{}
	}

	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT en.id, en.label, e."outputJson", e."createdAt"
		FROM "executionNodes" en
		LEFT JOIN executions e ON e."nodeId" = en.id AND e.status = 'COMPLETED'
		WHERE en."executionId" = $1 AND en.label = ANY($2)
		ORDER BY e."createdAt" ASC NULLS LAST
	`, executionID, names)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// label -> list of output values in time order
	type execOut struct {
		output any
		at     time.Time
	}
	nodeMap := map[string][]execOut{}
	for rows.Next() {
		var (
			id, label string
			out       []byte
			at        *time.Time
		)
		if err := rows.Scan(&id, &label, &out, &at); err != nil {
			return "", err
		}
		if len(out) == 0 {
			continue
		}
		var parsed any
		if err := json.Unmarshal(out, &parsed); err != nil {
			parsed = string(out)
		}
		// If stored as a JSON string of JSON, unwrap once
		if s, ok := parsed.(string); ok {
			var nested any
			if err := json.Unmarshal([]byte(s), &nested); err == nil {
				parsed = nested
			}
		}
		t := time.Time{}
		if at != nil {
			t = *at
		}
		nodeMap[label] = append(nodeMap[label], execOut{output: parsed, at: t})
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	parsedCode := code
	for _, r := range refs {
		execs := nodeMap[r.name]
		values := make([]any, 0)

		switch r.method {
		case "currentItem", "lastItem":
			if len(execs) > 0 {
				values = append(values, execs[len(execs)-1].output)
			}
		case "firstItem":
			if len(execs) > 0 {
				values = append(values, execs[0].output)
			}
		case "all":
			for _, e := range execs {
				values = append(values, e.output)
			}
		}

		var replacement string
		switch len(values) {
		case 0:
			replacement = "null"
		case 1:
			b, err := json.Marshal(values[0])
			if err != nil {
				replacement = "null"
			} else {
				replacement = string(b)
			}
		default:
			b, err := json.Marshal(values)
			if err != nil {
				replacement = "null"
			} else {
				replacement = string(b)
			}
		}
		parsedCode = strings.Replace(parsedCode, r.full, replacement, 1)
	}

	return parsedCode, nil
}

func MustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

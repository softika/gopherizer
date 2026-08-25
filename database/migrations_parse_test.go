package database

import (
	"regexp"
	"strings"
)

var (
	createTableRe = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_."]+)`)
	dropTableRe   = regexp.MustCompile(`(?is)DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([^;]+)`)
)

// splitGooseSections separates the Up and Down halves of a goose migration.
func splitGooseSections(body string) (up, down string) {
	idx := strings.Index(body, "-- +goose Down")
	if idx < 0 {
		return body, ""
	}
	return body[:idx], body[idx:]
}

func tablesInCreate(sql string) []string {
	var out []string
	for _, m := range createTableRe.FindAllStringSubmatch(sql, -1) {
		out = append(out, normalizeTable(m[1]))
	}
	return out
}

// tablesInDrop handles the multi-table form: DROP TABLE a, b, c;
func tablesInDrop(sql string) []string {
	var out []string
	for _, m := range dropTableRe.FindAllStringSubmatch(sql, -1) {
		for _, part := range strings.Split(m[1], ",") {
			part = strings.TrimSpace(part)
			part = strings.TrimSuffix(part, " CASCADE")
			part = strings.TrimSuffix(part, " RESTRICT")
			if t := normalizeTable(part); t != "" {
				out = append(out, t)
			}
		}
	}
	return out
}

func normalizeTable(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[i+1:]
	}
	return strings.ToLower(strings.Trim(s, `"`))
}

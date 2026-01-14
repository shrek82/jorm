package query

import (
	"strings"
)

type ClauseType int

const (
	SELECT ClauseType = iota
	FROM
	WHERE
	ORDERBY
	LIMIT
	OFFSET
	JOIN
)

type Clause struct {
	Type  ClauseType
	Value []any
}

func (c *Clause) Build() (string, []any) {
	switch c.Type {
	case SELECT:
		return "SELECT " + strings.Join(c.Value[0].([]string), ", "), nil
	case FROM:
		return "FROM " + c.Value[0].(string), nil
	case WHERE:
		conds := c.Value[0].([]string)
		if len(conds) == 0 {
			return "", nil
		}
		return "WHERE " + strings.Join(conds, " AND "), c.Value[1].([]any)
	case ORDERBY:
		return "ORDER BY " + strings.Join(c.Value[0].([]string), ", "), nil
	case LIMIT:
		return "LIMIT ?", []any{c.Value[0]}
	case OFFSET:
		return "OFFSET ?", []any{c.Value[0]}
	case JOIN:
		return strings.Join(c.Value[0].([]string), " "), nil
	}
	return "", nil
}

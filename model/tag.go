package model

import (
	"fmt"
	"strings"
)

// Tag represents parsed jorm tags
type Tag struct {
	Column       string
	PrimaryKey   bool
	AutoInc      bool
	Size         int
	Unique       bool
	NotNull      bool
	Default      string
	Fk           string
	AutoTime     bool
	AutoUpdate   bool
	RelationType string
	ForeignKey   string
	References   string
	JoinTable    string
	JoinFK       string
	JoinRef      string
	Type         string
}

// ParseTag parses the "jorm" tag string
func ParseTag(tagStr string) *Tag {
	tag := &Tag{}
	if tagStr == "" {
		return tag
	}

	// Support space, semicolon, comma as separators (but keep comma in parens)
	var sb strings.Builder
	inParen := false
	for _, r := range tagStr {
		switch r {
		case '(':
			inParen = true
			sb.WriteRune(r)
		case ')':
			inParen = false
			sb.WriteRune(r)
		case ';', ',':
			if inParen {
				sb.WriteRune(r)
			} else {
				sb.WriteRune(' ')
			}
		default:
			sb.WriteRune(r)
		}
	}
	tagStr = sb.String()
	parts := strings.Fields(tagStr) // Use Fields to split by whitespace and ignore empty strings

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, ":", 2)
		key := strings.ToLower(kv[0])
		var val string
		if len(kv) > 1 {
			val = kv[1]
		}

		subParts := strings.Split(val, ";")
		for i, subVal := range subParts {
			subVal = strings.TrimSpace(subVal)
			if i > 0 && len(subParts) > 1 {
				subKv := strings.SplitN(subVal, ":", 2)
				if len(subKv) > 1 {
					subKey := strings.ToLower(strings.TrimSpace(subKv[0]))
					subValue := strings.TrimSpace(subKv[1])

					switch subKey {
					case "relation":
						tag.RelationType = subValue
					case "references":
						tag.References = subValue
					case "join_table":
						tag.JoinTable = subValue
					case "join_fk":
						tag.JoinFK = subValue
					case "join_ref":
						tag.JoinRef = subValue
					}
				}
			}
		}

		switch key {
		case "column":
			tag.Column = strings.TrimSpace(subParts[0])
		case "pk":
			tag.PrimaryKey = true
		case "auto":
			tag.AutoInc = true
		case "unique":
			tag.Unique = true
		case "notnull":
			tag.NotNull = true
		case "size":
			if val != "" {
				fmt.Sscanf(val, "%d", &tag.Size)
			}
		case "default":
			tag.Default = strings.TrimSpace(subParts[0])
		case "fk":
			tag.Fk = strings.TrimSpace(subParts[0])
			tag.ForeignKey = strings.TrimSpace(subParts[0])
		case "auto_time":
			tag.AutoTime = true
		case "auto_update":
			tag.AutoUpdate = true
		case "type":
			tag.Type = strings.TrimSpace(subParts[0])
		case "many2many", "many_to_many":
			tag.RelationType = "many_to_many"
			if val != "" {
				tag.JoinTable = val
			}
		case "has_one", "has_many", "belongs_to":
			tag.RelationType = key
		case "foreignkey":
			tag.ForeignKey = strings.TrimSpace(subParts[0])
		case "references":
			tag.References = strings.TrimSpace(subParts[0])
		case "join_table":
			tag.JoinTable = strings.TrimSpace(subParts[0])
		case "join_fk":
			tag.JoinFK = strings.TrimSpace(subParts[0])
		case "join_ref":
			tag.JoinRef = strings.TrimSpace(subParts[0])
		case "relation":
			tag.RelationType = strings.TrimSpace(subParts[0])
		}
	}
	return tag
}

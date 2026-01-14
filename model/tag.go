package model

import (
	"strings"
)

// Tag represents parsed jorm tags
type Tag struct {
	Column     string
	PrimaryKey bool
	AutoInc    bool
	Size       int
	Unique     bool
	NotNull    bool
	Default    string
	Fk         string
	AutoTime   bool
	AutoUpdate bool
}

// ParseTag parses the "jorm" tag string
func ParseTag(tagStr string) *Tag {
	tag := &Tag{}
	if tagStr == "" {
		return tag
	}

	parts := strings.Split(tagStr, " ")
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

		switch key {
		case "column":
			tag.Column = val
		case "pk":
			tag.PrimaryKey = true
		case "auto":
			tag.AutoInc = true
		case "unique":
			tag.Unique = true
		case "notnull":
			tag.NotNull = true
		case "default":
			tag.Default = val
		case "fk":
			tag.Fk = val
		case "auto_time":
			tag.AutoTime = true
		case "auto_update":
			tag.AutoUpdate = true
		}
	}
	return tag
}

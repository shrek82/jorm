package tests

import (
	"testing"

	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/dialect"
)

func BenchmarkBuildSelect(b *testing.B) {
	d, _ := dialect.Get("mysql")
	builder := core.NewBuilder(d)
	builder.SetTable("users").
		Select("id", "name", "email", "age", "created_at").
		Where("age > ?", 18).
		Where("status = ?", "active").
		OrderBy("created_at DESC").
		Limit(10).
		Offset(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.BuildSelect()
	}
}

func BenchmarkBuildUpdate(b *testing.B) {
	d, _ := dialect.Get("mysql")
	builder := core.NewBuilder(d)
	builder.SetTable("users").
		Where("id = ?", 1)

	data := map[string]any{
		"name":       "New Name",
		"email":      "new@example.com",
		"updated_at": "2023-01-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.BuildUpdate(data)
	}
}

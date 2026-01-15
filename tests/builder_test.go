package tests

import (
	"strings"
	"testing"

	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/dialect"
)

func TestBuilder(t *testing.T) {
	d, _ := dialect.Get("sqlite3")

	t.Run("Select", func(t *testing.T) {
		b := core.NewBuilder(d)
		b.SetTable("users").Select("id", "name").Where("age > ?", 18).OrderBy("id DESC").Limit(10)
		sql, args := b.BuildSelect()

		expectedSQL := "SELECT id, name FROM `users` WHERE (age > ?) ORDER BY id DESC LIMIT ?"
		if sql != expectedSQL {
			t.Errorf("Expected SQL: %s\nGot: %s", expectedSQL, sql)
		}
		if len(args) != 2 || args[0] != 18 || args[1] != 10 {
			t.Errorf("Invalid args: %v", args)
		}
	})

	t.Run("Joins", func(t *testing.T) {
		b := core.NewBuilder(d)
		b.SetTable("orders").Alias("o").
			Select("o.id", "u.name").
			Joins("INNER JOIN users u ON u.id = o.user_id").
			Where("o.amount > ?", 100)
		sql, args := b.BuildSelect()

		if !strings.Contains(sql, "INNER JOIN users u ON u.id = o.user_id") {
			t.Errorf("Missing JOIN clause: %s", sql)
		}
		if !strings.Contains(sql, "FROM `orders` o") {
			t.Errorf("Missing table alias: %s", sql)
		}
		if args[0] != 100 {
			t.Errorf("Invalid args: %v", args)
		}
	})

	t.Run("JoinsInjection", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for SQL injection in Joins")
			}
		}()

		b := core.NewBuilder(d)
		b.Joins("INNER JOIN users; DROP TABLE users; --")
	})

	t.Run("WhereIn", func(t *testing.T) {
		b := core.NewBuilder(d)
		b.SetTable("users").WhereIn("id", []int{1, 2, 3})
		sql, args := b.BuildSelect()

		if !strings.Contains(sql, "id IN (?, ?, ?)") {
			t.Errorf("Invalid WhereIn SQL: %s", sql)
		}
		if len(args) != 3 {
			t.Errorf("Invalid args length: %d", len(args))
		}
	})

	t.Run("Update", func(t *testing.T) {
		b := core.NewBuilder(d)
		b.SetTable("users").Where("id = ?", 1)
		data := map[string]any{
			"name": "new_name",
			"age":  30,
		}
		sql, args := b.BuildUpdate(data)

		if !strings.Contains(sql, "UPDATE `users` SET") {
			t.Errorf("Invalid Update SQL: %s", sql)
		}
		if !strings.Contains(sql, "WHERE (id = ?)") {
			t.Errorf("Missing WHERE in Update: %s", sql)
		}
		// data columns are sorted in BuildUpdate
		if len(args) != 3 {
			t.Errorf("Invalid args length: %d", len(args))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		b := core.NewBuilder(d)
		b.SetTable("users").Where("id = ?", 1)
		sql, args := b.BuildDelete()

		if sql != "DELETE FROM `users` WHERE (id = ?)" {
			t.Errorf("Invalid Delete SQL: %s", sql)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("Invalid args: %v", args)
		}
	})
}

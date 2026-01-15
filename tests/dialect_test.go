package tests

import (
	"strings"
	"testing"

	"github.com/shrek82/jorm/dialect"
	"github.com/shrek82/jorm/model"
)

func TestDialects(t *testing.T) {
	drivers := []string{"mysql", "sqlite3", "postgres"}

	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			d, ok := dialect.Get(driver)
			if !ok {
				t.Fatalf("Dialect %s not found", driver)
			}

			// Test Quote
			quoted := d.Quote("user")
			if driver == "mysql" || driver == "sqlite3" {
				if quoted != "`user`" {
					t.Errorf("Expected `user`, got %s", quoted)
				}
			} else if driver == "postgres" {
				if quoted != "\"user\"" {
					t.Errorf("Expected \"user\", got %s", quoted)
				}
			}

			// Test Placeholder
			p1 := d.Placeholder(1)
			p2 := d.Placeholder(2)
			if driver == "postgres" {
				if p1 != "$1" || p2 != "$2" {
					t.Errorf("Expected $1, $2, got %s, %s", p1, p2)
				}
			} else {
				if p1 != "?" || p2 != "?" {
					t.Errorf("Expected ?, ?, got %s, %s", p1, p2)
				}
			}

			// Test InsertSQL
			sql, _ := d.InsertSQL("user", []string{"name", "age"})
			if !strings.Contains(sql, "INSERT INTO") {
				t.Errorf("Invalid InsertSQL: %s", sql)
			}

			// Test CreateTableSQL
			m, _ := model.GetModel(&TestUser{})
			createSQL, _ := d.CreateTableSQL(m)
			if !strings.Contains(createSQL, "CREATE TABLE") {
				t.Errorf("Invalid CreateTableSQL: %s", createSQL)
			}
		})
	}
}

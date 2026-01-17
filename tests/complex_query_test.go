package tests

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
)

type ComplexUser struct {
	ID   int64  `jorm:"pk;auto"`
	Name string `jorm:"column:name"`
	Age  int    `jorm:"column:age"`
}

type AgeGroup struct {
	Age   int `jorm:"column:age"`
	Count int `jorm:"column:user_count"`
}

func TestComplexQuery(t *testing.T) {
	dbFile := "complex_query_test.db"
	defer os.Remove(dbFile)

	db, err := core.Open("sqlite3", dbFile, nil)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = db.AutoMigrate(&ComplexUser{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Insert test data
	users := []ComplexUser{
		{Name: "User1", Age: 20},
		{Name: "User2", Age: 20},
		{Name: "User3", Age: 30},
		{Name: "User4", Age: 30},
		{Name: "User5", Age: 30},
		{Name: "User6", Age: 40},
	}
	for _, u := range users {
		_, err := db.Model(&u).Insert(&u)
		if err != nil {
			t.Fatalf("Failed to insert user: %v", err)
		}
	}

	t.Run("GroupBy", func(t *testing.T) {
		var results []AgeGroup
		err := db.Table("complex_user").
			Select("age", "COUNT(*) as user_count").
			GroupBy("age").
			OrderBy("age").
			Find(&results)
		if err != nil {
			t.Fatalf("GroupBy query failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 groups, got %d", len(results))
		}

		expected := []AgeGroup{
			{Age: 20, Count: 2},
			{Age: 30, Count: 3},
			{Age: 40, Count: 1},
		}

		for i, exp := range expected {
			if results[i].Age != exp.Age || results[i].Count != exp.Count {
				t.Errorf("Group %d: expected age %d count %d, got age %d count %d",
					i, exp.Age, exp.Count, results[i].Age, results[i].Count)
			}
		}
	})

	t.Run("GroupByHaving", func(t *testing.T) {
		var results []AgeGroup
		err := db.Table("complex_user").
			Select("age", "COUNT(*) as user_count").
			GroupBy("age").
			Having("COUNT(*) > ?", 1).
			OrderBy("age").
			Find(&results)
		if err != nil {
			t.Fatalf("GroupBy with Having query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 groups, got %d", len(results))
		}

		expected := []AgeGroup{
			{Age: 20, Count: 2},
			{Age: 30, Count: 3},
		}

		for i, exp := range expected {
			if results[i].Age != exp.Age || results[i].Count != exp.Count {
				t.Errorf("Group %d: expected age %d count %d, got age %d count %d",
					i, exp.Age, exp.Count, results[i].Age, results[i].Count)
			}
		}
	})
}

package tests

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
)

// HookUser supports hooks
type HookUser struct {
	ID    int64  `jorm:"pk;auto"`
	Name  string `jorm:"size:100"`
	Score int

	beforeInsertCalled bool
	afterFindCalled    bool
}

func (u *HookUser) TableName() string {
	return "hook_users"
}

func (u *HookUser) BeforeInsert() error {
	u.beforeInsertCalled = true
	if u.Name == "" {
		u.Name = "DefaultName"
	}
	return nil
}

func (u *HookUser) AfterFind() error {
	u.afterFindCalled = true
	return nil
}

// Embedded structs
type BaseInfo struct {
	CreatedBy string `jorm:"size:50"`
}

type Product struct {
	ID       int64  `jorm:"pk;auto"`
	Name     string `jorm:"size:100"`
	BaseInfo        // Embedded
	Price    float64
	Category string
}

func setupExtendedDB(t *testing.T) (*core.DB, func()) {
	t.Helper()
	dbFile := "extended_test.db"
	_ = os.Remove(dbFile)

	db, err := core.Open("sqlite3", dbFile, &core.Options{
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	err = db.AutoMigrate(&HookUser{}, &Product{})
	if err != nil {
		db.Close()
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	cleanup := func() {
		db.Close()
		_ = os.Remove(dbFile)
	}

	return db, cleanup
}

func TestHooks(t *testing.T) {
	db, cleanup := setupExtendedDB(t)
	defer cleanup()

	t.Run("BeforeInsert", func(t *testing.T) {
		user := &HookUser{Score: 100}
		_, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		if !user.beforeInsertCalled {
			t.Error("BeforeInsert hook was not called")
		}
		if user.Name != "DefaultName" {
			t.Errorf("Expected Name to be 'DefaultName' by hook, got '%s'", user.Name)
		}

		// Verify in DB
		var saved HookUser
		err = db.Model(&HookUser{}).Where("id = ?", user.ID).First(&saved)
		if err != nil {
			t.Fatalf("First failed: %v", err)
		}
		if saved.Name != "DefaultName" {
			t.Errorf("Expected saved Name to be 'DefaultName', got '%s'", saved.Name)
		}
	})

	t.Run("AfterFind", func(t *testing.T) {
		user := &HookUser{Name: "AfterFindUser"}
		_, _ = db.Model(user).Insert(user)

		var found HookUser
		err := db.Model(&HookUser{}).Where("name = ?", "AfterFindUser").First(&found)
		if err != nil {
			t.Fatalf("First failed: %v", err)
		}

		if !found.afterFindCalled {
			t.Error("AfterFind hook was not called")
		}
	})
}

func TestEmbeddedStructs(t *testing.T) {
	db, cleanup := setupExtendedDB(t)
	defer cleanup()

	t.Run("InsertAndFindEmbedded", func(t *testing.T) {
		p := &Product{
			Name:  "Laptop",
			Price: 999.99,
			BaseInfo: BaseInfo{
				CreatedBy: "Admin",
			},
		}
		_, err := db.Model(p).Insert(p)
		if err != nil {
			t.Fatalf("Insert embedded failed: %v", err)
		}

		var found Product
		err = db.Model(&Product{}).Where("name = ?", "Laptop").First(&found)
		if err != nil {
			t.Fatalf("First embedded failed: %v", err)
		}

		if found.CreatedBy != "Admin" {
			t.Errorf("Expected CreatedBy 'Admin', got '%s'", found.CreatedBy)
		}
	})
}

func TestGroupByHaving(t *testing.T) {
	db, cleanup := setupExtendedDB(t)
	defer cleanup()

	// Seed data
	products := []*Product{
		{Name: "P1", Category: "Electronics", Price: 100},
		{Name: "P2", Category: "Electronics", Price: 200},
		{Name: "P3", Category: "Books", Price: 50},
		{Name: "P4", Category: "Books", Price: 150},
	}
	for _, p := range products {
		_, _ = db.Model(p).Insert(p)
	}

	t.Run("GroupBy", func(t *testing.T) {
		type Result struct {
			Category string
			Total    float64 `jorm:"column:total"`
		}
		var results []Result
		// Using raw SQL for aggregation since high-level API might not support GroupBy yet
		// Let's check if db.Model().GroupBy exists
		err := db.Model(&Product{}).
			Select("category, SUM(price) as total").
			GroupBy("category").
			Find(&results)

		if err != nil {
			t.Fatalf("GroupBy failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 categories, got %d", len(results))
		}
	})

	t.Run("Having", func(t *testing.T) {
		type Result struct {
			Category string
			Total    float64 `jorm:"column:total"`
		}
		var results []Result
		err := db.Model(&Product{}).
			Select("category, SUM(price) as total").
			GroupBy("category").
			Having("total > ?", 200).
			Find(&results)

		if err != nil {
			t.Fatalf("Having failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 category with total > 200, got %d", len(results))
		}
		if results[0].Category != "Electronics" {
			t.Errorf("Expected category 'Electronics', got '%s'", results[0].Category)
		}
	})
}

func TestRawSQL(t *testing.T) {
	db, cleanup := setupExtendedDB(t)
	defer cleanup()

	// Seed data
	_, _ = db.Model(&Product{}).Insert(&Product{Name: "RawTest", Price: 50})

	t.Run("QueryRaw", func(t *testing.T) {
		var p Product
		err := db.Raw("SELECT * FROM product WHERE name = ?", "RawTest").Scan(&p)
		if err != nil {
			t.Fatalf("Raw SQL query failed: %v", err)
		}
		if p.Name != "RawTest" {
			t.Errorf("Expected Name 'RawTest', got '%s'", p.Name)
		}
	})

	t.Run("ExecRaw", func(t *testing.T) {
		_, err := db.Raw("UPDATE product SET price = ? WHERE name = ?", 75.0, "RawTest").Exec()
		if err != nil {
			t.Fatalf("Raw SQL exec failed: %v", err)
		}

		var p Product
		_ = db.Model(&Product{}).Where("name = ?", "RawTest").First(&p)
		if p.Price != 75.0 {
			t.Errorf("Expected price 75.0, got %f", p.Price)
		}
	})
}

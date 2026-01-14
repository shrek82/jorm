package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
	_ "github.com/shrek82/jorm/dialect"
)

type User struct {
	ID        int64     `jorm:"pk auto"`
	Name      string    `jorm:"size:100 notnull"`
	Email     string    `jorm:"size:100 unique"`
	Age       int       `jorm:"default:0"`
	CreatedAt time.Time `jorm:"auto_time"`
	UpdatedAt time.Time `jorm:"auto_update"`
}

type Order struct {
	ID     int64 `jorm:"pk auto"`
	UserID int64 `jorm:"column:user_id"`
	Amount float64
}

type Address struct {
	ID     int64  `jorm:"pk auto"`
	UserID int64  `jorm:"column:user_id"`
	City   string `jorm:"size:100"`
}

// Hooks for User
func (u *User) BeforeInsert() error {
	fmt.Println("BeforeInsert hook called")
	return nil
}

func (u *User) AfterFind() error {
	fmt.Println("AfterFind hook called")
	return nil
}

func setupTestDB(t *testing.T) (*core.DB, func()) {
	t.Helper()

	dbFile := "test.db"
	_ = os.Remove(dbFile)

	db, err := core.Open("sqlite3", dbFile, &core.Options{
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	err = db.AutoMigrate(&User{})
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

func TestIntegration(t *testing.T) {
	t.Run("AutoMigrate", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		var users []User
		err := db.Model(&User{}).Find(&users)
		if err != nil {
			t.Fatalf("Query after AutoMigrate failed: %v", err)
		}
	})

	t.Run("Insert", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		user := &User{
			Name:  "Alice",
			Email: "alice@example.com",
			Age:   25,
		}
		newID, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if newID == 0 {
			t.Fatal("Insert ID should not be 0")
		}

		var alice User
		err = db.Model(&User{}).Where("id = ?", newID).First(&alice)
		if err != nil {
			t.Fatalf("Find after insert failed: %v", err)
		}
		if alice.Name != "Alice" {
			t.Errorf("Expected name Alice, got %s", alice.Name)
		}
	})

	t.Run("FindOne", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		user := &User{
			Name:  "Alice",
			Email: "alice@example.com",
			Age:   25,
		}
		id, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert before find failed: %v", err)
		}

		var alice User
		err = db.Model(&User{}).Where("id = ?", id).First(&alice)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}
		if alice.Name != "Alice" {
			t.Errorf("Expected name Alice, got %s", alice.Name)
		}
	})

	t.Run("Update", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		user := &User{
			Name:  "Alice",
			Email: "alice@example.com",
			Age:   25,
		}
		id, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert before update failed: %v", err)
		}

		var alice User
		err = db.Model(&User{}).Where("id = ?", id).First(&alice)
		if err != nil {
			t.Fatalf("Find before update failed: %v", err)
		}
		alice.Age = 26
		affected, err := db.Model(&alice).Where("id = ?", id).Update(&alice)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if affected != 1 {
			t.Errorf("Expected 1 row affected, got %d", affected)
		}
	})

	t.Run("FindAll", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		usersToInsert := []*User{
			{
				Name:  "User1",
				Email: "user1@example.com",
				Age:   20,
			},
			{
				Name:  "User2",
				Email: "user2@example.com",
				Age:   30,
			},
		}

		for _, u := range usersToInsert {
			_, err := db.Model(u).Insert(u)
			if err != nil {
				t.Fatalf("Insert before find all failed: %v", err)
			}
		}

		var users []User
		err := db.Model(&User{}).Find(&users)
		if err != nil {
			t.Fatalf("Find all failed: %v", err)
		}
		if len(users) != len(usersToInsert) {
			t.Errorf("Expected %d users, got %d", len(usersToInsert), len(users))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		user := &User{
			Name:  "Alice",
			Email: "alice@example.com",
			Age:   25,
		}
		id, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert before delete failed: %v", err)
		}

		affected, err := db.Model(&User{}).Where("id = ?", id).Delete()
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if affected != 1 {
			t.Errorf("Expected 1 row affected by delete, got %d", affected)
		}
		var deletedUser User
		err = db.Model(&User{}).Where("id = ?", id).Find(&deletedUser)
		if err == nil {
			t.Error("Expected error finding deleted user, but got none")
		}
	})

	t.Run("TransactionCommit", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		err := db.Transaction(func(tx *core.Tx) error {
			user1 := &User{Name: "TxUser1", Email: "tx1@example.com"}
			_, err := tx.Model(user1).Insert(user1)
			if err != nil {
				return err
			}

			user2 := &User{Name: "TxUser2", Email: "tx2@example.com"}
			_, err = tx.Model(user2).Insert(user2)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Transaction failed: %v", err)
		}

		var txUsers []User
		err = db.Model(&User{}).Where("name LIKE ?", "TxUser%").Find(&txUsers)
		if err != nil {
			t.Fatalf("Find tx users failed: %v", err)
		}
		if len(txUsers) != 2 {
			t.Errorf("Expected 2 tx users, got %d", len(txUsers))
		}
	})

	t.Run("TransactionRollback", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		err := db.Transaction(func(tx *core.Tx) error {
			user3 := &User{Name: "TxUser3", Email: "tx3@example.com"}
			_, err := tx.Model(user3).Insert(user3)
			if err != nil {
				return err
			}
			return fmt.Errorf("trigger rollback")
		})
		if err == nil || err.Error() != "trigger rollback" {
			t.Fatalf("Expected rollback error, got %v", err)
		}

		var txUser3 User
		err = db.Model(&User{}).Where("name = ?", "TxUser3").First(&txUser3)
		if err == nil {
			t.Error("Expected TxUser3 to not exist due to rollback")
		}
	})

	t.Run("BatchInsert", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		batchUsers := []*User{
			{Name: "Batch1", Email: "b1@example.com"},
			{Name: "Batch2", Email: "b2@example.com"},
		}
		count, err := db.Model(&User{}).BatchInsert(batchUsers)
		if err != nil {
			t.Fatalf("BatchInsert failed: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 rows affected by BatchInsert, got %d", count)
		}
	})

	t.Run("Join", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		err := db.AutoMigrate(&Order{})
		if err != nil {
			t.Fatalf("AutoMigrate Order failed: %v", err)
		}

		user := &User{
			Name:  "JoinUser",
			Email: "join@example.com",
			Age:   30,
		}
		userID, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert user for join failed: %v", err)
		}

		order := &Order{
			UserID: userID,
			Amount: 100.5,
		}
		_, err = db.Model(order).Insert(order)
		if err != nil {
			t.Fatalf("Insert order for join failed: %v", err)
		}

		type OrderWithUser struct {
			ID       int64   `jorm:"column:id"`
			UserID   int64   `jorm:"column:user_id"`
			Amount   float64 `jorm:"column:amount"`
			UserName string  `jorm:"column:user_name"`
		}

		var results []OrderWithUser
		err = db.Model(&Order{}).
			Select(
				"`order`.id as id",
				"`order`.user_id as user_id",
				"`order`.amount as amount",
				"`user`.name as user_name",
			).
			Join("user", "INNER", "`user`.id = `order`.user_id").
			Where("`user`.name = ?", "JoinUser").
			Find(&results)
		if err != nil {
			t.Fatalf("Join query failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Expected 1 joined record, got %d", len(results))
		}
		if results[0].UserName != "JoinUser" {
			t.Errorf("Expected UserName JoinUser, got %s", results[0].UserName)
		}
		if results[0].Amount != 100.5 {
			t.Errorf("Expected Amount 100.5, got %v", results[0].Amount)
		}
	})

	t.Run("MultipleJoin", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		err := db.AutoMigrate(&Order{}, &Address{})
		if err != nil {
			t.Fatalf("AutoMigrate Order and Address failed: %v", err)
		}

		user := &User{
			Name:  "JoinMultiUser",
			Email: "joinmulti@example.com",
			Age:   35,
		}
		userID, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert user for multi-join failed: %v", err)
		}

		order := &Order{
			UserID: userID,
			Amount: 200.5,
		}
		_, err = db.Model(order).Insert(order)
		if err != nil {
			t.Fatalf("Insert order for multi-join failed: %v", err)
		}

		address := &Address{
			UserID: userID,
			City:   "Shanghai",
		}
		_, err = db.Model(address).Insert(address)
		if err != nil {
			t.Fatalf("Insert address for multi-join failed: %v", err)
		}

		type OrderWithUserAndAddress struct {
			ID       int64   `jorm:"column:id"`
			UserID   int64   `jorm:"column:user_id"`
			Amount   float64 `jorm:"column:amount"`
			UserName string  `jorm:"column:user_name"`
			City     string  `jorm:"column:city"`
		}

		var results []OrderWithUserAndAddress
		err = db.Model(&Order{}).
			Select(
				"`order`.id as id",
				"`order`.user_id as user_id",
				"`order`.amount as amount",
				"`user`.name as user_name",
				"`address`.city as city",
			).
			Join("user", "INNER", "`user`.id = `order`.user_id").
			Join("address", "INNER", "`address`.user_id = `order`.user_id").
			Where("`user`.name = ?", "JoinMultiUser").
			Find(&results)
		if err != nil {
			t.Fatalf("Multiple join query failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Expected 1 multi-joined record, got %d", len(results))
		}
		if results[0].UserName != "JoinMultiUser" {
			t.Errorf("Expected UserName JoinMultiUser, got %s", results[0].UserName)
		}
		if results[0].Amount != 200.5 {
			t.Errorf("Expected Amount 200.5, got %v", results[0].Amount)
		}
		if results[0].City != "Shanghai" {
			t.Errorf("Expected City Shanghai, got %s", results[0].City)
		}
	})

	t.Run("Sum", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		users := []*User{
			{
				Name:  "User1",
				Email: "user1@example.com",
				Age:   10,
			},
			{
				Name:  "User2",
				Email: "user2@example.com",
				Age:   20,
			},
		}

		for _, u := range users {
			_, err := db.Model(u).Insert(u)
			if err != nil {
				t.Fatalf("Insert before sum failed: %v", err)
			}
		}

		sum, err := db.Model(&User{}).Sum("age")
		if err != nil {
			t.Fatalf("Sum failed: %v", err)
		}
		if sum != 30 {
			t.Errorf("Expected sum age 30, got %v", sum)
		}
	})

	t.Run("MultipleWhereAndSelect", func(t *testing.T) {
		db, cleanup := setupTestDB(t)
		defer cleanup()

		user := &User{
			Name:  "MultiUser",
			Email: "multi@example.com",
			Age:   40,
		}
		id, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Insert before multiple where/select failed: %v", err)
		}

		var result struct {
			ID    int64  `jorm:"column:id"`
			Name  string `jorm:"column:name"`
			Email string `jorm:"column:email"`
		}

		query := db.Model(&User{})
		query = query.Where("id = ?", id)
		query = query.Where("email = ?", "multi@example.com")
		query = query.Select("id")
		query = query.Select("name", "email")

		err = query.First(&result)
		if err != nil {
			t.Fatalf("MultipleWhereAndSelect query failed: %v", err)
		}
		if result.ID != id {
			t.Errorf("Expected ID %d, got %d", id, result.ID)
		}
		if result.Name != "MultiUser" {
			t.Errorf("Expected Name MultiUser, got %s", result.Name)
		}
		if result.Email != "multi@example.com" {
			t.Errorf("Expected Email multi@example.com, got %s", result.Email)
		}
	})
}

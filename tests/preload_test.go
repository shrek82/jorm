package tests

import (
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shrek82/jorm/core"
)

type PreloadUser struct {
	ID        int64           `jorm:"pk auto"`
	Name      string          `jorm:"size:100"`
	Email     string          `jorm:"size:100 unique"`
	Age       int             `jorm:"default:0"`
	CreatedAt time.Time       `jorm:"auto_time"`
	UpdatedAt time.Time       `jorm:"auto_update"`
	Orders    []PreloadOrder  `jorm:"fk:UserID;relation:has_many"`
	Profile   *PreloadProfile `jorm:"fk:UserID;relation:has_one"`
	Roles     []PreloadRole   `jorm:"many_many:preload_user_role;join_fk:user_id;join_ref:role_id"`
}

type PreloadOrder struct {
	ID        int64            `jorm:"pk auto"`
	UserID    int64            `jorm:"fk:PreloadUser.ID"`
	Amount    float64          `jorm:"notnull"`
	Status    string           `jorm:"size:20 default:'pending'"`
	CreatedAt time.Time        `jorm:"auto_time"`
	UpdatedAt time.Time        `jorm:"auto_update"`
	User      *PreloadUser     `jorm:"fk:UserID;relation:belongs_to"`
	Products  []PreloadProduct `jorm:"many_many:preload_order_product;join_fk:order_id;join_ref:product_id"`
}

type PreloadProduct struct {
	ID        int64     `jorm:"pk auto"`
	Name      string    `jorm:"size:100 notnull"`
	Price     float64   `jorm:"notnull"`
	CreatedAt time.Time `jorm:"auto_time"`
	UpdatedAt time.Time `jorm:"auto_update"`
}

type PreloadProfile struct {
	ID        int64     `jorm:"pk auto"`
	UserID    int64     `jorm:"fk:PreloadUser.ID unique"`
	Bio       string    `jorm:"size:500"`
	CreatedAt time.Time `jorm:"auto_time"`
	UpdatedAt time.Time `jorm:"auto_update"`
}

type PreloadRole struct {
	ID        int64     `jorm:"pk auto"`
	Name      string    `jorm:"size:50 unique"`
	CreatedAt time.Time `jorm:"auto_time"`
	UpdatedAt time.Time `jorm:"auto_update"`
}

type PreloadUserRole struct {
	UserID int64 `jorm:"fk:PreloadUser.ID"`
	RoleID int64 `jorm:"fk:PreloadRole.ID"`
}

type PreloadOrderProduct struct {
	OrderID   int64 `jorm:"fk:PreloadOrder.ID"`
	ProductID int64 `jorm:"fk:PreloadProduct.ID"`
}

func setupPreloadDB(t *testing.T) *core.DB {
	db, err := core.Open("sqlite3", ":memory:", &core.Options{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&PreloadUser{}, &PreloadOrder{}, &PreloadProduct{}, &PreloadProfile{}, &PreloadRole{}, &PreloadUserRole{}, &PreloadOrderProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func cleanupPreloadDB(db *core.DB) {
	db.Exec("DELETE FROM preload_order_product")
	db.Exec("DELETE FROM preload_user_role")
	db.Exec("DELETE FROM preload_products")
	db.Exec("DELETE FROM preload_orders")
	db.Exec("DELETE FROM preload_profiles")
	db.Exec("DELETE FROM preload_roles")
	db.Exec("DELETE FROM preload_users")
}

func TestPreloadHasMany(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   25,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	for i := 0; i < 3; i++ {
		order := &PreloadOrder{
			UserID: userID,
			Amount: float64(i+1) * 100,
			Status: "completed",
		}
		_, err := db.Model(order).Insert(order)
		if err != nil {
			t.Fatalf("Failed to insert order: %v", err)
		}
	}

	var users []PreloadUser
	err = db.Model(&PreloadUser{}).
		Preload("Orders").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with preload: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	t.Logf("User found: %+v", users[0])
	t.Logf("User Orders field: %+v", users[0].Orders)
	t.Logf("User Orders length: %d", len(users[0].Orders))

	if len(users[0].Orders) != 3 {
		t.Fatalf("Expected 3 orders, got %d", len(users[0].Orders))
	}

	for _, order := range users[0].Orders {
		if order.UserID != userID {
			t.Errorf("Order UserID mismatch: expected %d, got %d", userID, order.UserID)
		}
	}
}

func TestPreloadBelongsTo(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   30,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	order := &PreloadOrder{
		UserID: userID,
		Amount: 200.0,
		Status: "pending",
	}
	orderID, err := db.Model(order).Insert(order)
	if err != nil {
		t.Fatalf("Failed to insert order: %v", err)
	}

	var orders []PreloadOrder
	err = db.Model(&PreloadOrder{}).
		Preload("User").
		Find(&orders)
	if err != nil {
		t.Fatalf("Failed to find orders with preload: %v", err)
	}

	if len(orders) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(orders))
	}

	if orders[0].User == nil {
		t.Fatal("Expected User to be loaded, got nil")
	}

	if orders[0].User.ID != userID {
		t.Errorf("User ID mismatch: expected %d, got %d", userID, orders[0].User.ID)
	}

	if orders[0].ID != orderID {
		t.Errorf("Order ID mismatch: expected %d, got %d", orderID, orders[0].ID)
	}
}

func TestPreloadHasOne(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Charlie",
		Email: "charlie@example.com",
		Age:   28,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	profile := &PreloadProfile{
		UserID: userID,
		Bio:    "Software engineer",
	}
	_, err = db.Model(profile).Insert(profile)
	if err != nil {
		t.Fatalf("Failed to insert profile: %v", err)
	}

	var users []PreloadUser
	err = db.Model(&PreloadUser{}).
		Preload("Profile").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with preload: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	if users[0].Profile == nil {
		t.Fatal("Expected Profile to be loaded, got nil")
	}

	if users[0].Profile.UserID != userID {
		t.Errorf("Profile UserID mismatch: expected %d, got %d", userID, users[0].Profile.UserID)
	}
}

func TestPreloadManyToMany(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "David",
		Email: "david@example.com",
		Age:   35,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	roles := []string{"admin", "editor", "viewer"}
	var roleIDs []int64

	for _, roleName := range roles {
		role := &PreloadRole{
			Name: roleName,
		}
		roleID, err := db.Model(role).Insert(role)
		if err != nil {
			t.Fatalf("Failed to insert role: %v", err)
		}
		roleIDs = append(roleIDs, roleID)

		userRole := &PreloadUserRole{
			UserID: userID,
			RoleID: roleID,
		}
		_, err = db.Model(userRole).Insert(userRole)
		if err != nil {
			t.Fatalf("Failed to insert user_role: %v", err)
		}
	}

	var users []PreloadUser
	err = db.Model(&PreloadUser{}).
		Preload("Roles").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with preload: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	if len(users[0].Roles) != 3 {
		t.Fatalf("Expected 3 roles, got %d", len(users[0].Roles))
	}

	for i, role := range users[0].Roles {
		if role.ID != roleIDs[i] {
			t.Errorf("Role ID mismatch at index %d: expected %d, got %d", i, roleIDs[i], role.ID)
		}
	}
}

func TestPreloadNested(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Eve",
		Email: "eve@example.com",
		Age:   27,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	order := &PreloadOrder{
		UserID: userID,
		Amount: 150.0,
		Status: "pending",
	}
	orderID, err := db.Model(order).Insert(order)
	if err != nil {
		t.Fatalf("Failed to insert order: %v", err)
	}

	for i := 0; i < 2; i++ {
		product := &PreloadProduct{
			Name:  "Product " + string(rune('A'+i)),
			Price: float64(i+1) * 50,
		}
		productID, err := db.Model(product).Insert(product)
		if err != nil {
			t.Fatalf("Failed to insert product: %v", err)
		}

		orderProduct := &PreloadOrderProduct{
			OrderID:   orderID,
			ProductID: productID,
		}
		_, err = db.Model(orderProduct).Insert(orderProduct)
		if err != nil {
			t.Fatalf("Failed to insert order_product: %v", err)
		}
	}

	var users []PreloadUser
	err = db.Model(&PreloadUser{}).
		Preload("Orders.Products").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with nested preload: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	if len(users[0].Orders) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(users[0].Orders))
	}

	if len(users[0].Orders[0].Products) != 2 {
		t.Fatalf("Expected 2 products, got %d", len(users[0].Orders[0].Products))
	}
}

func TestPreloadWithConditions(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Frank",
		Email: "frank@example.com",
		Age:   32,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	for i := 0; i < 5; i++ {
		order := &PreloadOrder{
			UserID: userID,
			Amount: float64(i+1) * 100,
			Status: []string{"pending", "completed", "pending", "cancelled", "completed"}[i],
		}
		_, err := db.Model(order).Insert(order)
		if err != nil {
			t.Fatalf("Failed to insert order: %v", err)
		}
	}

	var users []PreloadUser
	err = db.Model(&PreloadUser{}).
		PreloadWith("Orders", func(q *core.Query) {
			q.Where("status = ?", "pending").
				OrderBy("created_at DESC").
				Limit(2)
		}).
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with conditional preload: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	if len(users[0].Orders) != 2 {
		t.Fatalf("Expected 2 pending orders, got %d", len(users[0].Orders))
	}

	for _, order := range users[0].Orders {
		if order.Status != "pending" {
			t.Errorf("Expected order status to be 'pending', got '%s'", order.Status)
		}
	}
}

func TestPreloadMultiple(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Grace",
		Email: "grace@example.com",
		Age:   29,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	for i := 0; i < 2; i++ {
		order := &PreloadOrder{
			UserID: userID,
			Amount: float64(i+1) * 100,
			Status: "completed",
		}
		_, err := db.Model(order).Insert(order)
		if err != nil {
			t.Fatalf("Failed to insert order: %v", err)
		}
	}

	profile := &PreloadProfile{
		UserID: userID,
		Bio:    "Designer",
	}
	_, err = db.Model(profile).Insert(profile)
	if err != nil {
		t.Fatalf("Failed to insert profile: %v", err)
	}

	var users []PreloadUser
	err = db.Model(&PreloadUser{}).
		Preload("Orders").
		Preload("Profile").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with multiple preloads: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	if len(users[0].Orders) != 2 {
		t.Fatalf("Expected 2 orders, got %d", len(users[0].Orders))
	}

	if users[0].Profile == nil {
		t.Fatal("Expected Profile to be loaded, got nil")
	}
}

func TestPreloadEmpty(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	var users []PreloadUser
	err := db.Model(&PreloadUser{}).
		Preload("Orders").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users: %v", err)
	}

	if len(users) != 0 {
		t.Fatalf("Expected 0 users, got %d", len(users))
	}
}

func TestPreloadNonExistentRelation(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	var users []PreloadUser
	err := db.Model(&PreloadUser{}).
		Preload("NonExistent").
		Find(&users)
	if err == nil {
		t.Fatal("Expected error for non-existent relation, got nil")
	}
}

func TestPreloadPartialNoRelations(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user1 := &PreloadUser{
		Name:  "Henry",
		Email: "henry@example.com",
		Age:   31,
	}
	userID1, err := db.Model(user1).Insert(user1)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	user2 := &PreloadUser{
		Name:  "Ivy",
		Email: "ivy@example.com",
		Age:   26,
	}
	_, err = db.Model(user2).Insert(user2)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	order := &PreloadOrder{
		UserID: userID1,
		Amount: 200.0,
		Status: "pending",
	}
	_, err = db.Model(order).Insert(order)
	if err != nil {
		t.Fatalf("Failed to insert order: %v", err)
	}

	var users []PreloadUser
	err = db.Model(&PreloadUser{}).
		Preload("Orders").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with preload: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}

	if len(users[0].Orders) != 1 {
		t.Errorf("Expected 1 order for user %s, got %d", users[0].Name, len(users[0].Orders))
	}

	if len(users[1].Orders) != 0 {
		t.Errorf("Expected 0 orders for user %s, got %d", users[1].Name, len(users[1].Orders))
	}
}

func TestJoinsInner(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Jack",
		Email: "jack@example.com",
		Age:   33,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	order := &PreloadOrder{
		UserID: userID,
		Amount: 250.0,
		Status: "completed",
	}
	_, err = db.Model(order).Insert(order)
	if err != nil {
		t.Fatalf("Failed to insert order: %v", err)
	}

	type OrderWithUserInner struct {
		PreloadOrder
		UserName string `jorm:"column:user_name"`
	}

	var results []OrderWithUserInner
	err = db.Model(&PreloadOrder{}).
		Select("preload_order.*", "preload_user.name as user_name").
		Joins("preload_user", "INNER", "preload_user.id = preload_order.user_id").
		Find(&results)
	if err != nil {
		t.Fatalf("Failed to find orders with joins: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].UserName != "Jack" {
		t.Errorf("Expected user name 'Jack', got '%s'", results[0].UserName)
	}
}

func TestJoinsLeft(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Kate",
		Email: "kate@example.com",
		Age:   24,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	order := &PreloadOrder{
		UserID: userID,
		Amount: 180.0,
		Status: "pending",
	}
	_, err = db.Model(order).Insert(order)
	if err != nil {
		t.Fatalf("Failed to insert order: %v", err)
	}

	type OrderWithUser struct {
		PreloadOrder
		UserName string `jorm:"column:user_name"`
	}

	var results []OrderWithUser
	err = db.Model(&PreloadOrder{}).
		Select("preload_order.*", "preload_user.name as user_name").
		Joins("preload_user", "LEFT", "preload_user.id = preload_order.user_id").
		Find(&results)
	if err != nil {
		t.Fatalf("Failed to find orders with left join: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].UserName != "Kate" {
		t.Errorf("Expected user name 'Kate', got '%s'", results[0].UserName)
	}
}

func TestPreloadFirst(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	user := &PreloadUser{
		Name:  "Leo",
		Email: "leo@example.com",
		Age:   30,
	}
	userID, err := db.Model(user).Insert(user)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	order := &PreloadOrder{
		UserID: userID,
		Amount: 300.0,
		Status: "completed",
	}
	_, err = db.Model(order).Insert(order)
	if err != nil {
		t.Fatalf("Failed to insert order: %v", err)
	}

	var foundUser PreloadUser
	err = db.Model(&PreloadUser{}).
		Preload("Orders").
		Where("id = ?", userID).
		First(&foundUser)
	if err != nil {
		t.Fatalf("Failed to find user with preload: %v", err)
	}

	if foundUser.ID != userID {
		t.Errorf("Expected user ID %d, got %d", userID, foundUser.ID)
	}

	if len(foundUser.Orders) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(foundUser.Orders))
	}
}

func TestPreloadPerformance(t *testing.T) {
	db := setupPreloadDB(t)
	defer db.Close()
	defer cleanupPreloadDB(db)

	userCount := 10
	ordersPerUser := 5

	for i := 0; i < userCount; i++ {
		user := &PreloadUser{
			Name:  "User " + string(rune('A'+i)),
			Email: "user" + string(rune('0'+i)) + "@example.com",
			Age:   20 + i,
		}
		userID, err := db.Model(user).Insert(user)
		if err != nil {
			t.Fatalf("Failed to insert user: %v", err)
		}

		for j := 0; j < ordersPerUser; j++ {
			order := &PreloadOrder{
				UserID: userID,
				Amount: float64(j+1) * 100,
				Status: "completed",
			}
			_, err := db.Model(order).Insert(order)
			if err != nil {
				t.Fatalf("Failed to insert order: %v", err)
			}
		}
	}

	var users []PreloadUser
	err := db.Model(&PreloadUser{}).
		Preload("Orders").
		Find(&users)
	if err != nil {
		t.Fatalf("Failed to find users with preload: %v", err)
	}

	if len(users) != userCount {
		t.Fatalf("Expected %d users, got %d", userCount, len(users))
	}

	for i, user := range users {
		if len(user.Orders) != ordersPerUser {
			t.Errorf("User %d: Expected %d orders, got %d", i, ordersPerUser, len(user.Orders))
		}
	}
}

package tests

import (
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDebugPreloadHasOne(t *testing.T) {
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
	fmt.Printf("Inserted User ID: %d\n", userID)

	profile := &PreloadProfile{
		UserID: userID,
		Bio:    "Software engineer",
	}
	_, err = db.Model(profile).Insert(profile)
	if err != nil {
		t.Fatalf("Failed to insert profile: %v", err)
	}
	fmt.Printf("Inserted Profile with UserID: %d\n", userID)

	// Verify profile exists in DB
	var profiles []PreloadProfile
	err = db.Model(&PreloadProfile{}).Where("user_id = ?", userID).Find(&profiles)
	if err != nil {
		t.Fatalf("Failed to find profile: %v", err)
	}
	if len(profiles) == 0 {
		t.Fatalf("Profile not found in DB")
	}
	fmt.Printf("Found Profile: %+v\n", profiles[0])

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

	fmt.Printf("DEBUG: users[0] addr: %p\n", &users[0])
	fmt.Printf("User[0].Profile: %v\n", users[0].Profile)

	if users[0].Profile == nil {
		t.Fatal("Expected Profile to be loaded, got nil")
	}

	if users[0].Profile.UserID != userID {
		t.Errorf("Profile UserID mismatch: expected %d, got %d", userID, users[0].Profile.UserID)
	}
}

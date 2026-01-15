package core

// BeforeInserter is the interface for the BeforeInsert hook.
// It is called before a record is inserted into the database.
type BeforeInserter interface{ BeforeInsert() error }

// AfterInserter is the interface for the AfterInsert hook.
// It is called after a record is successfully inserted into the database.
type AfterInserter interface{ AfterInsert(id int64) error }

// BeforeUpdater is the interface for the BeforeUpdate hook.
// It is called before a record is updated in the database.
type BeforeUpdater interface{ BeforeUpdate() error }

// AfterUpdater is the interface for the AfterUpdate hook.
// It is called after a record is successfully updated in the database.
type AfterUpdater interface{ AfterUpdate() error }

// BeforeDeleter is the interface for the BeforeDelete hook.
// It is called before a record is deleted from the database.
type BeforeDeleter interface{ BeforeDelete() error }

// AfterDeleter is the interface for the AfterDelete hook.
// It is called after a record is successfully deleted from the database.
type AfterDeleter interface{ AfterDelete() error }

// AfterFinder is the interface for the AfterFind hook.
// It is called after a record is retrieved from the database.
type AfterFinder interface{ AfterFind() error }

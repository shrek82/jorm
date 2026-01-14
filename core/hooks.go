package core

type BeforeInserter interface{ BeforeInsert() error }
type AfterInserter interface{ AfterInsert(id int64) error }
type BeforeUpdater interface{ BeforeUpdate() error }
type AfterUpdater interface{ AfterUpdate() error }
type BeforeDeleter interface{ BeforeDelete() error }
type AfterDeleter interface{ AfterDelete() error }
type AfterFinder interface{ AfterFind() error }


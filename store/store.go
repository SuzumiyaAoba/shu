// Package store provides the persistence layer for the shu RSS aggregator.
//
// It provides a concrete SQLite implementation via [SQLiteStore], which
// satisfies the [core.Store] interface. The package imports [core] only for
// model types ([core.Feed], [core.Entry], [core.EntryFilter]) and must never
// depend on the cmd package.
package store

package store

import (
	"errors"
	"log"

	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

// MigrateOwnerID backfills Group/Card OwnerID for rows created before
// v0.2.28 (A.5 R1). AutoMigrate adds the column with zero default; this
// function finds the existing admin user and stamps every owner_id=0 row
// with that ID.
//
// Idempotent: a second call finds zero rows matching `owner_id = 0` and
// silently returns. Safe to call from both the boot bootstrap chain and
// after backup.go's restore (the restored backup may predate v0.2.28 and
// arrive with owner_id=0 fields).
//
// The admin lookup uses `username = "admin"` rather than `id = 1` because
// id assignment isn't guaranteed (someone could have manually edited the
// DB or restored a backup where admin is id 7).
//
// If no admin user exists (fresh install before bootstrapAdmin runs),
// returns nil and skips — there's nothing to backfill against. Once
// bootstrapAdmin runs, a subsequent MigrateOwnerID call would backfill
// any straggler rows created during the gap (in practice: none, because
// new rows from R1 handlers stamp OwnerID at create time).
func MigrateOwnerID(db *gorm.DB) error {
	var admin model.User
	err := db.Where("username = ?", "admin").First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No admin yet — nothing to migrate. bootstrapAdmin runs before
		// this in main.go, so the only path here is "fresh install with
		// MOON_ADMIN_PASSWORD unset", which has zero rows to migrate
		// anyway.
		return nil
	}
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		groups := tx.Model(&model.Group{}).Where("owner_id = 0").Update("owner_id", admin.ID)
		if groups.Error != nil {
			return groups.Error
		}
		if groups.RowsAffected > 0 {
			log.Printf("R1 migration: stamped owner_id=%d on %d existing groups", admin.ID, groups.RowsAffected)
		}

		cards := tx.Model(&model.Card{}).Where("owner_id = 0").Update("owner_id", admin.ID)
		if cards.Error != nil {
			return cards.Error
		}
		if cards.RowsAffected > 0 {
			log.Printf("R1 migration: stamped owner_id=%d on %d existing cards", admin.ID, cards.RowsAffected)
		}
		return nil
	})
}

// engineCategoryDefault mirrors api.CategoryWeb. Duplicated as a literal
// because api imports store (backup.go), so store must not import api.
const engineCategoryDefault = "web"

// MigrateEngineCategories backfills SearchEngine.Category for rows created
// before v0.2.31. AutoMigrate adds the column to existing rows as '' (a column
// default only applies to inserts), which would otherwise land those engines in
// the frontend's "其它" catch-all group. Everything predating v0.2.31 was a
// general web engine, so '' → "web".
//
// Idempotent: the WHERE only ever matches unstamped rows, so a second call
// updates nothing. Engines the user later moves to another category are never
// touched — their category isn't ''.
//
// Returns the number of rows stamped. Safe to call from both the boot
// bootstrap chain and after backup.go's restore (a restored backup may predate
// v0.2.31 and arrive with empty category fields).
func MigrateEngineCategories(db *gorm.DB) (int64, error) {
	res := db.Model(&model.SearchEngine{}).
		Where("category IS NULL OR category = ?", "").
		Update("category", engineCategoryDefault)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		log.Printf("v0.2.31 migration: stamped category=%s on %d existing search engines", engineCategoryDefault, res.RowsAffected)
	}
	return res.RowsAffected, nil
}

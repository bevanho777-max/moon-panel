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

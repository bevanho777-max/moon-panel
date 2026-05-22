package store

import (
	"testing"

	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

// MigrateOwnerID is v0.2.28's first multi-user roadmap migration and the
// project's first Go test. These cases lock in the contract for the rest
// of the A.5 roadmap so R2/R3 can rely on backfill semantics.

func TestMigrateOwnerID_BackfillsAdminID(t *testing.T) {
	db := openTestDB(t)

	// admin user gets some id (not necessarily 1 — depends on table state).
	admin := model.User{Username: "admin", PasswordHash: "x"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	// Pre-v0.2.28 rows: owner_id is 0 (column default after AutoMigrate).
	g := model.Group{Name: "legacy", Sort: 10}
	if err := db.Create(&g).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}
	c := model.Card{GroupID: g.ID, Title: "legacy", URLInternal: "http://x"}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("seed card: %v", err)
	}

	if err := MigrateOwnerID(db); err != nil {
		t.Fatalf("MigrateOwnerID: %v", err)
	}

	var got model.Group
	db.First(&got, g.ID)
	if got.OwnerID != admin.ID {
		t.Errorf("group OwnerID: want %d, got %d", admin.ID, got.OwnerID)
	}
	var gotCard model.Card
	db.First(&gotCard, c.ID)
	if gotCard.OwnerID != admin.ID {
		t.Errorf("card OwnerID: want %d, got %d", admin.ID, gotCard.OwnerID)
	}
}

func TestMigrateOwnerID_Idempotent(t *testing.T) {
	db := openTestDB(t)

	admin := model.User{Username: "admin", PasswordHash: "x"}
	db.Create(&admin)
	g := model.Group{Name: "legacy"}
	db.Create(&g)

	// First run: backfills.
	if err := MigrateOwnerID(db); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// Second run: no-op (no rows match owner_id = 0).
	if err := MigrateOwnerID(db); err != nil {
		t.Fatalf("second run: %v", err)
	}

	var got model.Group
	db.First(&got, g.ID)
	if got.OwnerID != admin.ID {
		t.Errorf("after second run: want %d, got %d", admin.ID, got.OwnerID)
	}
}

func TestMigrateOwnerID_NoAdmin_SkipsCleanly(t *testing.T) {
	// Fresh install before bootstrapAdmin runs — no users in the table.
	// Migration should return nil, not error out, so the boot chain stays
	// fatal-free for empty installs.
	db := openTestDB(t)

	if err := MigrateOwnerID(db); err != nil {
		t.Fatalf("expected nil on empty user table, got: %v", err)
	}
}

func TestMigrateOwnerID_DoesNotOverwriteNonZero(t *testing.T) {
	// A row that already has an OwnerID (e.g. created on v0.2.28+ by the new
	// handlers) must not be touched — the WHERE owner_id = 0 clause
	// guarantees this, and this test pins the invariant.
	db := openTestDB(t)

	admin := model.User{Username: "admin", PasswordHash: "x"}
	db.Create(&admin)
	other := model.User{Username: "future-user", PasswordHash: "y"}
	db.Create(&other)

	g := model.Group{Name: "owned", OwnerID: other.ID}
	db.Create(&g)

	if err := MigrateOwnerID(db); err != nil {
		t.Fatalf("MigrateOwnerID: %v", err)
	}

	var got model.Group
	db.First(&got, g.ID)
	if got.OwnerID != other.ID {
		t.Errorf("owned group rewritten: want %d (preserved), got %d", other.ID, got.OwnerID)
	}
}

// openTestDB returns a fresh GORM connection backed by a per-test SQLite
// database under t.TempDir(). Registers an explicit Close so Windows can
// release the WAL-mode file lock before t.TempDir's auto-cleanup tries to
// rm the directory (otherwise the test reports a phantom cleanup failure
// even when the assertions passed).
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("openTestDB: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

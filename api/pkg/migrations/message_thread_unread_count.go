package migrations

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/stacktrace"
	"gorm.io/gorm"
)

// MigrateMessageThreadUnreadCount migrates message thread unread count schema.
func MigrateMessageThreadUnreadCount(db *gorm.DB) error {
	if err := db.AutoMigrate(&entities.MessageThread{}, &entities.MessageThreadUnreadItem{}); err != nil {
		return stacktrace.Propagate(err, "cannot migrate message thread unread count schema")
	}

	if !db.Migrator().HasColumn("message_threads", "is_read") {
		return nil
	}

	if err := db.Table("message_threads").
		Where("is_read = ?", false).
		Where("unread_count = ?", 0).
		Update("unread_count", 1).Error; err != nil {
		return stacktrace.Propagate(err, "cannot backfill message thread unread counts")
	}

	if err := db.Migrator().DropColumn("message_threads", "is_read"); err != nil {
		return stacktrace.Propagate(err, "cannot drop legacy message thread is_read column")
	}

	return nil
}

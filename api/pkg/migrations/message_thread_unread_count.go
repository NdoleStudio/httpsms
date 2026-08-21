package migrations

import (
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/stacktrace"
	"gorm.io/gorm"
)

const messageThreadConversationIndexName = "idx_message_threads_conversation"

type messageThreadConversationIndex struct {
	UserID  entities.UserID `gorm:"column:user_id;uniqueIndex:idx_message_threads_conversation"`
	Owner   string          `gorm:"column:owner;uniqueIndex:idx_message_threads_conversation"`
	Contact string          `gorm:"column:contact;uniqueIndex:idx_message_threads_conversation"`
}

func (messageThreadConversationIndex) TableName() string {
	return "message_threads"
}

// MigrateMessageThreadUnreadCount migrates message thread unread count schema.
func MigrateMessageThreadUnreadCount(db *gorm.DB) error {
	if err := db.AutoMigrate(&entities.MessageThread{}, &entities.MessageThreadUnreadItem{}); err != nil {
		return stacktrace.Propagate(err, "cannot migrate message thread unread count schema")
	}

	needsConversationIndex := !db.Migrator().HasIndex("message_threads", messageThreadConversationIndexName)
	if needsConversationIndex {
		var duplicates []messageThreadConversationIndex
		if err := db.
			Model(&entities.MessageThread{}).
			Select("user_id", "owner", "contact").
			Group("user_id, owner, contact").
			Having("COUNT(*) > ?", 1).
			Limit(1).
			Find(&duplicates).
			Error; err != nil {
			return stacktrace.Propagate(err, "cannot check duplicate message thread conversations")
		}
		if len(duplicates) != 0 {
			return stacktrace.NewError(
				"cannot create unique message thread conversation index: duplicate message thread conversations exist",
			)
		}
	}

	if db.Migrator().HasColumn("message_threads", "is_read") {
		if err := db.Table("message_threads").
			Where("is_read = ?", false).
			Where("unread_count = ?", 0).
			Update("unread_count", 1).Error; err != nil {
			return stacktrace.Propagate(err, "cannot backfill message thread unread counts")
		}

		if err := db.Migrator().DropColumn("message_threads", "is_read"); err != nil {
			return stacktrace.Propagate(err, "cannot drop legacy message thread is_read column")
		}
	}

	if !needsConversationIndex {
		return nil
	}

	if err := db.Migrator().CreateIndex(&messageThreadConversationIndex{}, messageThreadConversationIndexName); err != nil {
		return stacktrace.Propagate(err, "cannot create unique message thread conversation index")
	}

	return nil
}

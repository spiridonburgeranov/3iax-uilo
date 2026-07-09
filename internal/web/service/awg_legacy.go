package service

import (
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"

	"gorm.io/gorm"
)

func DeleteLegacyAwgClientsByEmails(tx *gorm.DB, emails ...string) (int64, error) {
	if len(emails) == 0 {
		return 0, nil
	}
	trimmed := make([]string, 0, len(emails))
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email != "" {
			trimmed = append(trimmed, email)
		}
	}
	if len(trimmed) == 0 {
		return 0, nil
	}
	if tx == nil {
		tx = database.GetDB()
	}
	result := tx.Where("email IN ?", trimmed).Delete(&model.AwgClient{})
	return result.RowsAffected, result.Error
}

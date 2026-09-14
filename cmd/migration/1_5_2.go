package migration

import (
	"github.com/alireza0/s-ui/util"

	"gorm.io/gorm"
)

// to1_5_2 hashes any plaintext admin passwords stored in the users table so a
// leaked database no longer exposes credentials in the clear.
func to1_5_2(tx *gorm.DB) error {
	type userRow struct {
		Id       uint
		Password string
	}
	var users []userRow
	if err := tx.Raw("SELECT id, password FROM users").Scan(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if u.Password == "" || util.IsHashedPassword(u.Password) {
			continue
		}
		hashed, err := util.HashPassword(u.Password)
		if err != nil {
			return err
		}
		if err := tx.Exec("UPDATE users SET password = ? WHERE id = ?", hashed, u.Id).Error; err != nil {
			return err
		}
	}
	return nil
}

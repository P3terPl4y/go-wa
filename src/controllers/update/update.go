package update

import (
	"App/src/global"

	"golang.org/x/crypto/bcrypt"
)

func UpdateBotBlocked(botID int, blocked bool) error {
	_, err := global.ConfigDB.Exec(`UPDATE bots SET blocked = $1 WHERE id = $2`, blocked, botID)
	return err
}

func UpdateBotPaymentStatus(botID int, status string) error {
	_, err := global.ConfigDB.Exec(`UPDATE bots SET payment_status = $1 WHERE id = $2`, status, botID)
	return err
}
func UpdateUserPassword(userID int, newPassword string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = global.ConfigDB.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, string(hashed), userID)
	return err
}
func UpdateUserPhone(userID int, newPhone string) error {
    if newPhone == "" {
        return fmt.Errorf("el teléfono no puede estar vacío")
    }
    var count int
    err := configDB.QueryRow(`SELECT COUNT(*) FROM users WHERE phone = $1 AND id != $2`, newPhone, userID).Scan(&count)
    if err != nil {
        return err
    }
    if count > 0 {
        return fmt.Errorf("el número de teléfono ya está registrado por otro usuario")
    }
    _, err = configDB.Exec(`UPDATE users SET phone = $1 WHERE id = $2`, newPhone, userID)
    return err
}

package store

import (
	"log"

	"api-escalada/db"
)

func SetToken(token string, userID int) {
	if db.DB == nil {
		log.Printf("warning: db not initialized, token not persisted")
		return
	}
	if _, err := db.DB.Exec(`INSERT INTO auth_tokens (token, usuario_id) VALUES ($1, $2) ON CONFLICT (token) DO UPDATE SET usuario_id = EXCLUDED.usuario_id, created_at = NOW()`, token, userID); err != nil {
		log.Printf("warning: could not persist auth token: %v", err)
	}
}

func GetUserID(token string) (int, bool) {
	if db.DB == nil {
		return 0, false
	}
	var userID int
	if err := db.DB.QueryRow(`SELECT usuario_id FROM auth_tokens WHERE token = $1`, token).Scan(&userID); err != nil {
		return 0, false
	}
	return userID, true
}

func DeleteToken(token string) {
	if db.DB == nil {
		return
	}
	if _, err := db.DB.Exec(`DELETE FROM auth_tokens WHERE token = $1`, token); err != nil {
		log.Printf("warning: could not delete auth token: %v", err)
	}
}

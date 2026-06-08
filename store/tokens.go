package store

import "sync"

var tokens sync.Map

func SetToken(token string, userID int) {
	tokens.Store(token, userID)
}

func GetUserID(token string) (int, bool) {
	val, ok := tokens.Load(token)
	if !ok {
		return 0, false
	}
	return val.(int), true
}

func DeleteToken(token string) {
	tokens.Delete(token)
}

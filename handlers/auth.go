package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"api-escalada/db"
	"api-escalada/store"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Nombre   string `json:"nombre"`
}

type loginResponse struct {
	Token string   `json:"token"`
	User  userInfo `json:"user"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	var user userInfo
	err := db.DB.QueryRow(
		"SELECT id, username, nombre FROM usuarios WHERE username=$1 AND password=$2",
		req.Username, req.Password,
	).Scan(&user.ID, &user.Username, &user.Nombre)
	if err != nil {
		http.Error(w, `{"error":"credenciales invalidas"}`, http.StatusUnauthorized)
		return
	}
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	store.SetToken(token, user.ID)
	json.NewEncoder(w).Encode(loginResponse{Token: token, User: user})
}

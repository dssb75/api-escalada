package handlers

import (
	"encoding/json"
	"net/http"

	"api-escalada/db"
)

type Equipo struct {
	ID          int    `json:"id"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
	ImagenURL   string `json:"imagen_url"`
	Disponible  bool   `json:"disponible"`
}

func GetEquipos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.DB.Query("SELECT id, nombre, descripcion, imagen_url, disponible FROM equipos ORDER BY id")
	if err != nil {
		http.Error(w, `{"error":"error fetching equipos"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var equipos []Equipo
	for rows.Next() {
		var e Equipo
		rows.Scan(&e.ID, &e.Nombre, &e.Descripcion, &e.ImagenURL, &e.Disponible)
		equipos = append(equipos, e)
	}
	if equipos == nil {
		equipos = []Equipo{}
	}
	json.NewEncoder(w).Encode(equipos)
}

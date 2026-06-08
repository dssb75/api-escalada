package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"api-escalada/db"
	"api-escalada/middleware"
)

type ReservaEquipo struct {
	ID           int    `json:"id"`
	EquipoID     int    `json:"equipo_id"`
	EquipoNombre string `json:"equipo_nombre"`
	Fecha        string `json:"fecha"`
	Estado       string `json:"estado"`
}

type ReservaHorario struct {
	ID     int    `json:"id"`
	Fecha  string `json:"fecha"`
	Hora   string `json:"hora"`
	Estado string `json:"estado"`
	Mine   bool   `json:"mine"`
}

func ReservasEquipo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := r.Context().Value(middleware.UserIDKey).(int)

	switch r.Method {
	case http.MethodGet:
		rows, err := db.DB.Query(`
			SELECT re.id, re.equipo_id, e.nombre, re.fecha::text, re.estado
			FROM reservas_equipo re JOIN equipos e ON e.id = re.equipo_id
			WHERE re.usuario_id = $1 ORDER BY re.fecha DESC
		`, userID)
		if err != nil {
			http.Error(w, `{"error":"error fetching reservas"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var reservas []ReservaEquipo
		for rows.Next() {
			var re ReservaEquipo
			rows.Scan(&re.ID, &re.EquipoID, &re.EquipoNombre, &re.Fecha, &re.Estado)
			reservas = append(reservas, re)
		}
		if reservas == nil {
			reservas = []ReservaEquipo{}
		}
		json.NewEncoder(w).Encode(reservas)

	case http.MethodPost:
		var req struct {
			EquipoID int    `json:"equipo_id"`
			Fecha    string `json:"fecha"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		var id int
		err := db.DB.QueryRow(
			`INSERT INTO reservas_equipo (usuario_id, equipo_id, fecha) VALUES ($1,$2,$3) RETURNING id`,
			userID, req.EquipoID, req.Fecha,
		).Scan(&id)
		if err != nil {
			http.Error(w, `{"error":"error creating reserva"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int{"id": id})

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, `{"error":"invalid reserva id"}`, http.StatusBadRequest)
			return
		}
		res, err := db.DB.Exec(`DELETE FROM reservas_equipo WHERE id = $1 AND usuario_id = $2`, id, userID)
		if err != nil {
			http.Error(w, `{"error":"error deleting reserva"}`, http.StatusInternalServerError)
			return
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			http.Error(w, `{"error":"reserva no encontrada"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func ReservasHorario(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := r.Context().Value(middleware.UserIDKey).(int)

	switch r.Method {
	case http.MethodGet:
		fecha := r.URL.Query().Get("fecha")
		var reservas []ReservaHorario
		if fecha != "" {
			rows, err := db.DB.Query(`
				SELECT id, fecha::text, hora, estado, (usuario_id = $1) as mine
				FROM reservas_horario WHERE fecha = $2 ORDER BY hora
			`, userID, fecha)
			if err != nil {
				http.Error(w, `{"error":"error fetching horarios"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			for rows.Next() {
				var rh ReservaHorario
				rows.Scan(&rh.ID, &rh.Fecha, &rh.Hora, &rh.Estado, &rh.Mine)
				reservas = append(reservas, rh)
			}
		} else {
			rows, err := db.DB.Query(`
				SELECT id, fecha::text, hora, estado, true
				FROM reservas_horario WHERE usuario_id = $1 ORDER BY fecha DESC, hora
			`, userID)
			if err != nil {
				http.Error(w, `{"error":"error fetching horarios"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			for rows.Next() {
				var rh ReservaHorario
				rows.Scan(&rh.ID, &rh.Fecha, &rh.Hora, &rh.Estado, &rh.Mine)
				reservas = append(reservas, rh)
			}
		}
		if reservas == nil {
			reservas = []ReservaHorario{}
		}
		json.NewEncoder(w).Encode(reservas)

	case http.MethodPost:
		var req struct {
			Fecha string `json:"fecha"`
			Hora  string `json:"hora"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		isBusiness, err := isBusinessDay(req.Fecha)
		if err != nil {
			http.Error(w, `{"error":"fecha invalida"}`, http.StatusBadRequest)
			return
		}
		if !isBusiness {
			http.Error(w, `{"error":"solo se permiten reservas en dias habiles (lunes a viernes)"}`, http.StatusBadRequest)
			return
		}
		if !isValidHora(req.Hora) {
			http.Error(w, `{"error":"horario invalido, servicio activo de 08:00 a 22:00"}`, http.StatusBadRequest)
			return
		}
		var existing int
		db.DB.QueryRow(
			`SELECT COUNT(*) FROM reservas_horario WHERE fecha = $1 AND hora = $2`,
			req.Fecha, req.Hora,
		).Scan(&existing)
		if existing > 0 {
			http.Error(w, `{"error":"slot ya reservado"}`, http.StatusConflict)
			return
		}
		var id int
		err = db.DB.QueryRow(
			`INSERT INTO reservas_horario (usuario_id, fecha, hora) VALUES ($1,$2,$3) RETURNING id`,
			userID, req.Fecha, req.Hora,
		).Scan(&id)
		if err != nil {
			http.Error(w, `{"error":"error creating reserva"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int{"id": id})

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, `{"error":"invalid reserva id"}`, http.StatusBadRequest)
			return
		}
		res, err := db.DB.Exec(`DELETE FROM reservas_horario WHERE id = $1 AND usuario_id = $2`, id, userID)
		if err != nil {
			http.Error(w, `{"error":"error deleting reserva"}`, http.StatusInternalServerError)
			return
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			http.Error(w, `{"error":"reserva no encontrada"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func isBusinessDay(fecha string) (bool, error) {
	t, err := time.Parse("2006-01-02", fecha)
	if err != nil {
		return false, err
	}
	w := t.Weekday()
	return w >= time.Monday && w <= time.Friday, nil
}

func isValidHora(hora string) bool {
	t, err := time.Parse("15:04", hora)
	if err != nil {
		return false
	}
	minutes := t.Hour()*60 + t.Minute()
	return t.Minute() == 0 && minutes >= 8*60 && minutes <= 22*60
}

package handlers

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"api-escalada/db"
	"api-escalada/middleware"
	"api-escalada/modules/mailer"
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
			Email    string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		if _, err := mail.ParseAddress(strings.TrimSpace(req.Email)); err != nil {
			http.Error(w, `{"error":"correo electronico invalido"}`, http.StatusBadRequest)
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
		emailSent := sendReservaEquipoEmail(userID, id, req.EquipoID, req.Fecha, strings.TrimSpace(req.Email))
		if emailSent != nil {
			log.Printf("warning: equipment reservation email not sent: %v", emailSent)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id, "email_sent": emailSent == nil})

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
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		if _, err := mail.ParseAddress(strings.TrimSpace(req.Email)); err != nil {
			http.Error(w, `{"error":"correo electronico invalido"}`, http.StatusBadRequest)
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
		emailSent := sendReservaHorarioEmail(userID, id, req.Fecha, req.Hora, strings.TrimSpace(req.Email))
		if emailSent != nil {
			log.Printf("warning: schedule reservation email not sent: %v", emailSent)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id, "email_sent": emailSent == nil})

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

type userContact struct {
	Nombre string
	Email  string
}

func getUserContact(userID int) (userContact, error) {
	var contact userContact
	err := db.DB.QueryRow(
		`SELECT nombre, COALESCE(email, '') FROM usuarios WHERE id = $1`,
		userID,
	).Scan(&contact.Nombre, &contact.Email)
	if err != nil {
		return userContact{}, err
	}
	if contact.Email == "" {
		return userContact{}, fmt.Errorf("usuario sin correo registrado")
	}
	return contact, nil
}

func sendReservaEquipoEmail(userID, reservaID, equipoID int, fecha, recipientEmail string) error {
	contact, err := getUserContact(userID)
	if err != nil {
		return err
	}
	var equipoNombre string
	if err := db.DB.QueryRow(`SELECT nombre FROM equipos WHERE id = $1`, equipoID).Scan(&equipoNombre); err != nil {
		return err
	}
	subject := fmt.Sprintf("Confirmacion de reserva de equipo #%d", reservaID)
	body := buildReservationEmailBody("Reserva de equipo confirmada", map[string]string{
		"Reserva":   fmt.Sprintf("#%d", reservaID),
		"Usuario":   contact.Nombre,
		"Correo":    contact.Email,
		"Equipo":    equipoNombre,
		"Fecha":     fecha,
		"Estado":    "activa",
		"Notificar": recipientEmail,
	})
	return mailer.SendHTML(recipientEmail, subject, body)
}

func sendReservaHorarioEmail(userID, reservaID int, fecha, hora, recipientEmail string) error {
	contact, err := getUserContact(userID)
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("Confirmacion de reserva de horario #%d", reservaID)
	body := buildReservationEmailBody("Reserva de horario confirmada", map[string]string{
		"Reserva":   fmt.Sprintf("#%d", reservaID),
		"Usuario":   contact.Nombre,
		"Correo":    contact.Email,
		"Fecha":     fecha,
		"Hora":      hora,
		"Estado":    "activa",
		"Notificar": recipientEmail,
	})
	return mailer.SendHTML(recipientEmail, subject, body)
}

func buildReservationEmailBody(title string, details map[string]string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><body style='font-family:Arial,sans-serif;background:#f6f8fb;color:#111827;padding:24px'>")
	b.WriteString("<div style='max-width:640px;margin:0 auto;background:#ffffff;border:1px solid #e5e7eb;border-radius:12px;padding:24px'>")
	b.WriteString("<h2 style='margin:0 0 12px;font-size:20px;color:#0f172a'>" + html.EscapeString(title) + "</h2>")
	b.WriteString("<p style='margin:0 0 16px;color:#334155'>Tu reserva quedo registrada con esta informacion:</p>")
	b.WriteString("<table style='width:100%;border-collapse:collapse'>")
	for label, value := range details {
		b.WriteString("<tr><td style='padding:8px 0;border-bottom:1px solid #e5e7eb;font-weight:700;color:#0f172a'>" + html.EscapeString(label) + "</td><td style='padding:8px 0;border-bottom:1px solid #e5e7eb;color:#334155'>" + html.EscapeString(value) + "</td></tr>")
	}
	b.WriteString("</table>")
	b.WriteString("<p style='margin:16px 0 0;color:#64748b;font-size:12px'>EscaLab</p>")
	b.WriteString("</div></body></html>")
	return b.String()
}

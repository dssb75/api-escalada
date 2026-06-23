package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(connStr string) error {
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	if err = DB.Ping(); err != nil {
		return err
	}
	migrate()
	return nil
}

func migrate() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS usuarios (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			nombre VARCHAR(100) NOT NULL,
			email VARCHAR(150),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS equipos (
			id SERIAL PRIMARY KEY,
			nombre VARCHAR(100) UNIQUE NOT NULL,
			descripcion TEXT,
			imagen_url VARCHAR(500),
			disponible BOOLEAN DEFAULT true
		)`,
		`CREATE TABLE IF NOT EXISTS reservas_equipo (
			id SERIAL PRIMARY KEY,
			usuario_id INTEGER REFERENCES usuarios(id),
			equipo_id INTEGER REFERENCES equipos(id),
			fecha DATE NOT NULL,
			estado VARCHAR(20) DEFAULT 'activa',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS reservas_horario (
			id SERIAL PRIMARY KEY,
			usuario_id INTEGER REFERENCES usuarios(id),
			fecha DATE NOT NULL,
			hora VARCHAR(10) NOT NULL,
			estado VARCHAR(20) DEFAULT 'activa',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS auth_tokens (
			token VARCHAR(128) PRIMARY KEY,
			usuario_id INTEGER REFERENCES usuarios(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			log.Printf("Migration warning: %v", err)
		}
	}
	if _, err := DB.Exec(`ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS email VARCHAR(150)`); err != nil {
		log.Printf("Migration warning: %v", err)
	}
	seedData()
	log.Println("Database ready")
}

func seedData() {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM usuarios").Scan(&count)
	if count == 0 {
		DB.Exec(`INSERT INTO usuarios (username, password, nombre, email) VALUES ('admin', 'admin123', 'Administrador', 'admin@escalada.local')`)
		DB.Exec(`INSERT INTO usuarios (username, password, nombre, email) VALUES ('escalador', 'escalar123', 'Carlos Rueda', 'carlos.rueda@escalada.local')`)
		log.Println("Usuarios seeded")
	}
	DB.Exec(`UPDATE usuarios SET email = 'admin@escalada.local' WHERE username = 'admin' AND (email IS NULL OR email = '')`)
	DB.Exec(`UPDATE usuarios SET email = 'carlos.rueda@escalada.local' WHERE username = 'escalador' AND (email IS NULL OR email = '')`)
	DB.QueryRow("SELECT COUNT(*) FROM equipos").Scan(&count)
	equipos := []struct{ nombre, desc, img string }{
		{"Arnés de escalada", "Arnés de seguridad homologado para escalada en roca y muro", "https://images.pexels.com/photos/1733056/pexels-photo-1733056.jpeg?auto=compress&cs=tinysrgb&w=900&h=600&dpr=1"},
		{"Casco de escalada", "Casco de protección certificado contra impactos verticales y laterales", "https://images.pexels.com/photos/1365425/pexels-photo-1365425.jpeg?auto=compress&cs=tinysrgb&w=900&h=600&dpr=1"},
		{"Zapatillas de escalada", "Calzado técnico con suela de goma especial para máximo agarre", "https://images.pexels.com/photos/1598505/pexels-photo-1598505.jpeg?auto=compress&cs=tinysrgb&w=900&h=600&dpr=1"},
		{"Cuerda dinámica 60m", "Cuerda dinámica de 10.2mm para escalada deportiva y en roca", "https://images.pexels.com/photos/8961013/pexels-photo-8961013.jpeg?auto=compress&cs=tinysrgb&w=900&h=600&dpr=1"},
		{"Chalk bag", "Bolsa de magnesio con cinturón ajustable para mejor agarre", "https://images.pexels.com/photos/674010/pexels-photo-674010.jpeg?auto=compress&cs=tinysrgb&w=900&h=600&dpr=1"},
		{"Set mosquetones", "Set de 6 mosquetones con seguro de rosca certificados UIAA", "https://images.pexels.com/photos/159581/carabiner-gear-ropes-climbing-159581.jpeg?auto=compress&cs=tinysrgb&w=900&h=600&dpr=1"},
	}
	if count == 0 {
		for _, e := range equipos {
			DB.Exec(`INSERT INTO equipos (nombre, descripcion, imagen_url) VALUES ($1, $2, $3)`, e.nombre, e.desc, e.img)
		}
		log.Println("Equipos seeded")
	}
	for _, e := range equipos {
		DB.Exec(`UPDATE equipos SET descripcion = $1, imagen_url = $2 WHERE nombre = $3`, e.desc, e.img, e.nombre)
	}
}

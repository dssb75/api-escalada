package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(connStr string) error {
	var err error

	// FIX RDS: forzar SSL
	connStr = connStr + " sslmode=require"

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
}

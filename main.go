package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"api-escalada/db"
	"api-escalada/handlers"
	"api-escalada/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	if err := db.Init(connStr); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/api/auth/login", handlers.Login)
	mux.Handle("/api/equipos", middleware.Auth(http.HandlerFunc(handlers.GetEquipos)))
	mux.Handle("/api/reservas/equipo", middleware.Auth(http.HandlerFunc(handlers.ReservasEquipo)))
	mux.Handle("/api/reservas/horario", middleware.Auth(http.HandlerFunc(handlers.ReservasHorario)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := middleware.CORS(mux)
	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

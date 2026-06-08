# api-escalada

Backend mínimo en Go con PostgreSQL.

## Requisitos
- Go 1.21+
- PostgreSQL

## Configuración
Copia `.env` y ajusta los valores:
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=escalada
PORT=8080
```

## Ejecutar
```bash
go run .
```

## Endpoints
- `GET /health` — estado del servidor

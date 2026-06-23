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
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASSWORD=your-password
EMAIL_FROM=reservas@example.com
EMAIL_PROVIDER=SMTP
```

Las reservas de equipo y horario envian un correo de confirmacion al usuario con los datos de la reserva. Si no configuras SMTP, la reserva sigue guardandose pero el correo no se envia.

## Ejecutar
```bash
go run .
```

## Endpoints
- `GET /health` — estado del servidor

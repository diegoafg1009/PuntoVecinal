# Project punto-vecinal

API en Go de Punto Vecinal, la plataforma de reportes ciudadanos de la Cooperativa Primavera (Comas, Lima). Los vecinos reportan problemas del barrio con foto y ubicación, apoyan los reportes de otros con "También me afecta", y los gestores de casos los derivan a la entidad responsable y actualizan su estado.

- **HTTP:** router chi con la capa generada por oapi-codegen por módulo (`internal/<módulo>/api/generated.go`), a partir del contrato dividido en `openapi/` (raíz del repositorio) y empaquetado con Redocly CLI en `openapi/dist/openapi.yaml`.
- **Datos:** Supabase Postgres (PostGIS y pgvector), con acceso generado por sqlc desde las migraciones de `supabase/migrations/`.
- **Autenticación:** JWT de Supabase Auth, verificado en cada petición protegida (ver [Autenticación](#autenticación)).
- **Archivos y embeddings:** DigitalOcean Spaces (compatible con S3) y Voyage AI.
- **Configuración:** variables de entorno con respaldo en `.env` (viper); ver `.env.example`.

## Autenticación

El inicio de sesión, el registro y el refresco del token los resuelve Supabase Auth directamente desde el cliente (incluida la sesión anónima del vecino sin cuenta); la API no tiene endpoints para eso. La API solo valida el token:

- Toda operación del contrato (`openapi/`) exige `Authorization: Bearer <access_token>` salvo las que declaran `security: []` (hoy, `GET /healthz`). Una operación nueva queda protegida por defecto.
- El token se verifica con las claves públicas de `SUPABASE_JWKSURL` (firma, expiración, emisor `SUPABASE_URL/auth/v1` y audiencia `authenticated`). Un token ausente, vencido o inválido responde `401`.
- `GET /me` devuelve `{ "id": "<uuid>", "isAnonymous": bool }` del token verificado. El `id` no cambia al refrescar el token.
- `SUPABASE_URL` y `SUPABASE_JWKSURL` son obligatorias: sin ellas la API no arranca (valores locales en `.env.example`).
- Para probar en local con la pila de Supabase en marcha (`make supabase-start`), con `ANON_KEY` de `make supabase-status`:

  ```bash
  curl -s -X POST http://127.0.0.1:54321/auth/v1/signup -H "apikey: $ANON_KEY" -H "Content-Type: application/json" -d '{}'   # devuelve access_token
  curl -s localhost:8080/me -H "Authorization: Bearer $ACCESS_TOKEN"
  ```

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```

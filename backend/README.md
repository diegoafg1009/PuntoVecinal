# Project punto-vecinal

API en Go de Punto Vecinal, la plataforma de reportes ciudadanos de la Cooperativa Primavera (Comas, Lima). Los vecinos reportan problemas del barrio con foto y ubicación, apoyan los reportes de otros con "También me afecta", y los gestores de casos los derivan a la entidad responsable y actualizan su estado.

- **HTTP:** router chi con la capa generada por oapi-codegen a partir del contrato `api/openapi.yaml`.
- **Datos:** Supabase Postgres (PostGIS y pgvector), con acceso generado por sqlc desde las migraciones de `supabase/migrations/`.
- **Autenticación:** JWT de Supabase Auth, verificado en cada petición.
- **Archivos y embeddings:** DigitalOcean Spaces (compatible con S3) y Voyage AI.
- **Configuración:** variables de entorno con respaldo en `.env` (viper); ver `.env.example`.

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

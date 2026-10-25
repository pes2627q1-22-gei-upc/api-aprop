# Instruccions per als assistents de codi — AProp! (backend)

AProp! és la plataforma de FiveGuys (PES, FIB–UPC) per descobrir i planificar activitats culturals a
Catalunya a partir de les Dades Obertes de la Generalitat. Aquest repositori és l'**API backend en Go** (`api-aprop`).
El client mòbil Flutter és a `../frontend-aprop` (amb el seu propi `AGENTS.md`).

## Abans de començar

1. Llegeix `docs/ai/estat.md` (sprint, història i branca actives). És curt.
2. Per a codi i disseny, llegeix `docs/architecture.md`. Carrega la resta de `docs/ai/` **només si la tasca ho necessita** (índex a `docs/ai/README.md`).
3. **No llegeixis `docs/source/*.pdf`** si no t'ho demanen explícitament: `docs/ai/` ja en conté el resum.

## Límit d'acció (regla de l'equip)

- **Revisió humana obligatòria:** cap assistent no fa `git commit`, `git push`, merge ni tag sense una
  confirmació explícita en aquell mateix torn. Proposa el missatge de commit (`Tasca #<ref> <Títol>: <descripció>`) i deixa els canvis per revisar.
- Indica sempre què has modificat i què has pogut verificar. No presentis com a acabada una integració pendent ni mocks com si fossin reals.
- **Decisions pendents:** si fas un canvi sense estar-ne segur (requisit ambigu, contracte inexistent, dependència nova, valor deduït), registra'l a `docs/decisions-pendents.md` (DP-NNN), marca el codi amb `TODO(DP-NNN)` i digues-ho a la resposta. Consulta les DP obertes abans de tocar el que hi està relacionat.

## Stack del backend

- **Go 1.25** amb la biblioteca estàndard `net/http` (`http.ServeMux` amb patrons `MÈTODE /ruta/{param}`). Sense framework web extern (ADR-0001).
- **Base de dades:** Supabase (PostgreSQL 15+ gestionat al cloud) amb Row Level Security (RLS) habilitat (ADR-0004). Driver `pgx/v5` (`internal/drivers/postgres_driver`).
- **Consultes SQL:** `sqlc` v1.31.1 (`sqlc.yaml`, `internal/repositories/postgres/queries/*.sql` → `internal/repositories/postgres/sqlc/`).
- **Migracions:** `migrations/` incrustades amb `embed.FS` per al binari i `supabase/migrations/` per al cicle de vida de Supabase CLI.
- **IA i Chatbot:** Groq AI (Chat Completions amb function calling / tool calling) connectat amb el servidor MCP intern (`cmd/mcp`, ADR-0005).
- **Autenticació:** Firebase Auth (OIDC, validació de JWT amb Firebase Admin SDK / `firebase_client`).
- **Dades externes:** API Dades Obertes de la Generalitat de Catalunya (`gencat_client`), càrrega periòdica d'esdeveniments oficials.
- **Docker:** Imatge de dues etapes (build Go compilant estàticament `cmd/app` i `cmd/mcp`).

## Ordres habituals

```bash
# Execució local
go run ./cmd/app        # Servidor API REST principal (per defecte port 8080)
go run ./cmd/mcp        # Servidor MCP per a eines d'esdeveniments (port 8081)

# Format i anàlisi de codi
gofmt -s -w .
go vet ./...

# Proves unitàries i d'integració
go test ./...
go test -v -race ./...
go test ./internal/handlers/event_handler/...

# Generació de consultes tipades amb sqlc
# (Compte: 'models.go' i 'events.sql.go' tenen ajustos manuals, no regenerar a cegues)
sqlc generate

# Docker
docker build -t api-aprop .
docker run --env-file .env -p 8080:8080 api-aprop
```

## Arquitectura i codi

- **Arquitectura en capes / Ports i Adaptadors:**
  - `cmd/app/main.go`: Punt d'entrada. Carrega variables d'entorn (`.env`), connecta a la BD, executa migracions, instancia clients, repositoris, serveis i handlers, i arrenca el servidor `http.Server`.
  - `cmd/mcp/main.go`: Servidor MCP que exposa eines del catàleg per al xatbot.
  - `internal/router/router.go`: Registre de rutes passant per middleware global (`authMiddleware`).
  - `internal/handlers/<x>_handler/`: HTTP handlers. Parsegen paràmetres/JSON, validen entrades, criden el servei i responen usant `internal/handlers/response`.
  - `internal/services/<x>_service/`: Lògica de negoci. Defineixen interfície (`interface.go`) i implementació. No depenen d'HTTP.
  - `internal/repositories/<x>_repository/`: Interfície de persistència. Implementacions a `internal/repositories/postgres/`.
  - `internal/clients/<x>_client/`: Clients de serveis externs darrere d'interfícies (proves deterministes amb fakes/mocks).
  - `internal/models/<x>_model/`: Structs de domini purs amb tags JSON en **kebab-case**.
- **JSON sempre en kebab-case:** (`start-date`, `place-id`, `free`, `tickets-url`, etc.).
- **Gestió d'errors i respostes:** Centralitzada amb `internal/handlers/response`. Retornar codis HTTP adequats (`200 OK`, `201 Created`, `400 Bad Request`, `401 Unauthorized`, `404 Not Found`, `502 Bad Gateway`, etc.).
- **Transaccions:** Utilitza transaccions (`pgx.Tx`) sempre que una operació modifiqui múltiples taules o files correlacionades.
- Codi i identificadors en anglès; termes de domini segons `docs/ai/70-glossari.md`.

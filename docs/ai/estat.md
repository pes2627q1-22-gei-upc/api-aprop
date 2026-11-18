# Estat viu del Backend (`api-aprop`)

> **Quan llegir-ho:** sempre a l'inici d'una sessió de treball i en passar el relleu a un altre membre o assistent.
> S'actualitza a cada canvi significatiu d'estat.

## Sprint

- **Fase:** Sprint 1 (MVP)
- **Període:** 05/10/2026 – 28/10/2026
- **Objectiu:** Posada en marxa de la infraestructura base, ingesta de dades reals de la Generalitat, visualització d'esdeveniments (llista i mapa), registre/autenticació d'usuaris, inscripció bàsica a activitats i assistent cultural amb IA.

## En curs

| Taiga | Història / Tasca | Repo | Branca | Responsable | Estat |
|---|---|---|---|---|---|
| **#188** | Endpoint `PUT /events/{id}`: autoria, tipus `NoOficial`, estat actiu i R5 | `api-aprop` | `feature/modificar-esdeveniment` | Arnau Serra | Implementat amb tests unitaris; migració 000017 i SQL no provats contra BD; pendent de PR a `develop` (DP-034 oberta) |
| **#112** | Endpoints per a Mobilicat: `POST /api/v1/congestion/estimate` (API key, snake_case) | `api-aprop` | `feature/mobilicat-endpoints` | Arnau Serra | Implementat i provat amb tests unitaris; SQL no provat contra BD; pendent de PR a `develop` (DP-032 oberta) |
| **#116** | Endpoint `DELETE /users/me`: esborrat transaccional RGPD i baixa a Firebase Auth | `api-aprop` | `feature/delete-users` | Arnau Serra | Implementat amb tests unitaris; SQL no provat contra BD; PR a `develop` (DP-031 oberta) |
| **#246** | Unificar la identificació de l'usuari (Firebase UID → UUID) i els helpers d'IDs | `api-aprop` | `feature/unificar-usuari` | Arnau Serra | Pendent de PR a `develop` (DP-011 segueix oberta per a notificacions) |
| **#236** | Correccions dels endpoints d'esdeveniments (llistat, mapa, modificació, imatge, recomanats) | `api-aprop` | `Fix-esdeveniments` | Arnau Serra | PR oberta a `develop`, pendent de revisió (DP-026 oberta) |
| **#228** | Adaptar i estructurar la documentació al backend | `api-aprop` | `documentation` | Wenqiang He | Documentació sincronitzada, adaptada a Go/Supabase i en revisió |
| **#226** | Endpoints test d'interessos (US #27) | `api-aprop` | `feature/interest-test` | Wenqiang He | Branca creada, taules llestes al model de Supabase |
| **#186** | Endpoint `DELETE /events/{id}/enroll` (US #49) | `api-aprop` | `develop` | Wenqiang He | Assignada al Sprint 1 per completar el cicle d'inscripcions |
| **#165** | Endpoints de consulta de perfil (US #36) | `api-aprop` | `develop` | Wenqiang He | Assignada al Sprint 1 |
| **#119** | Endpoint edició de perfil (US #12) | `api-aprop` | `develop` | Wenqiang He | Assignada al Sprint 1 |

## Fet recentment (Sprint 1)

- **Release v1.0.0 (MVP):** Preparació de la primera release v1.0.0 del backend (`api-aprop`) cap a `main` amb el pipeline complet de CI/CD (validació de PRs, comprovació de build de binaris `cmd/app` i `cmd/mcp`, execució de tests, linting, anàlisi de vulnerabilitats i publicació d'imatge Docker a GHCR + GitHub Release en etiquetes `v*`).
- **#224 (Tasca Tancada):** Migració de base de dades PostgreSQL a Supabase amb migracions versionades (000001 a 000005) i Row Level Security (RLS) habilitat. Col·lecció Postman configurada.
- **#225 (Tasca Tancada):** Reimplementació del xatbot assistent amb Groq AI (tool calling) integrat amb el servidor MCP (`cmd/mcp`).
- **#78 / #80 (Tasques Tancades):** Integració d'endpoints per al mapa d'esdeveniments (`GET /events/map` i `GET /event/map`).
- **#88:** Backend per al filtratge d'esdeveniments i consulta de categories (`GET /categories`).
- **#7:** Sincronització d'usuaris entre Firebase Auth i Supabase (`POST /auth/sync`).
- **#221:** Endpoints de consulta del calendari / inscripció a esdeveniments (`POST /events/{id}/signup`, `GET /users/{userId}/events`).

## Propers passos

1. Finalitzar la branca `documentation` i preparar el commit per a revisió.
2. Iniciar la implementació de la tasca **#226** a la branca `feature/interest-test` (endpoints per al test de 25 preguntes i càlcul de pesos de categories a `UserPreferences`).

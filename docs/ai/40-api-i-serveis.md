# 40 · API i serveis

> **Quan llegir-ho:** en crear o consumir un endpoint, en integrar un servei extern o en tractar amb
> Talaia/Mobilicat. El contracte real és el codi del backend: si hi ha discrepància, mana el codi
> (`cmd/app/main.go`, `internal/router/router.go`, `internal/handlers/*`).
> **Font:** `incepcio-2.pdf` §9.5, §11 i l'estat real de `api-aprop@develop`.

## Endpoints existents (backend `develop`)

Totes les respostes i cossos de petició utilitzen claus JSON en **kebab-case**.

### Esdeveniments i Categories (`event_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `GET /events` | `EventHandler.ListEvents` | Llistat d'esdeveniments amb filtres per categoria, dates i cerca | Implementat |
| `GET /events/{id}` | `EventHandler.GetEvent` | Detall complet d'un esdeveniment per ID | Implementat |
| `POST /events` | `EventHandler.CreateEvent` | Creació d'esdeveniments no oficials | Implementat |
| `PUT /events/{id}` | `EventHandler.UpdateEvent` | Modificació d'un esdeveniment no oficial: només el creador (`403` si no ho és o si és oficial), només si l'estat és `Active` (`409` si no, p. ex. `Withdrawn`) i complint R5 contra la data de creació (`400`) | Implementat (DP-034) |
| `DELETE /events/{id}` | `EventHandler.DeleteEvent` | Esborrat d'un esdeveniment no oficial i de les dades que en depenen (només el creador; 204) | Implementat |
| `GET /events/map` | `EventHandler.ListMapEvents` | Esdeveniments amb coordenades i metadades bàsiques per al mapa | Implementat |
| `GET /categories` | `EventHandler.GetCategories` | Llistat de totes les categories del catàleg | Implementat |
| `GET /categories/{id}` | `EventHandler.GetCategory` | Detall d'una categoria específica | Implementat |

### Inscripcions i Agenda (`signup_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /events/{id}/signup` | `SignUpHandler.SignUp` | Inscripció d'un usuari a un esdeveniment | Implementat |
| `GET /users/{userId}/events` | `SignUpHandler.ListUserEvents` | Consulta de les activitats a les quals s'ha apuntat l'usuari (Agenda) | Implementat |

### Assistent Virtual i Xatbot (`assistant_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /assistant/messages` | `AssistantHandler.SendMessage` | Envia un missatge a l'assistent; orquestra Groq AI i eines MCP per retornar resposta i recomanacions | Implementat |
| `GET /assistant/conversation` | `AssistantHandler.GetConversation` | Recupera l'historial de la conversa activa de l'usuari | Implementat |

### Autenticació i Usuaris (`auth_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /auth/sync` | `AuthHandler.Sync` | Sincronitza l'usuari autenticat amb Firebase a la taula d'usuaris de Supabase | Implementat |

### Compte d'usuari (`user_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `DELETE /users/me` | `UserHandler.DeleteMyAccount` | Baixa del compte (RGPD): esborra en una transacció l'usuari, la seva configuració, els esdeveniments no oficials que ha creat (amb les dades que en depenen) i totes les files que el referencien; dins la transacció esborra el compte de Firebase Auth i, després del commit, l'avatar de S3 (best-effort). `204`; `401` sense identitat; `404` usuari inexistent; `502` si Firebase falla (no s'esborra res) | Implementat (DP-031) |

### Integració amb Mobilicat (`congestion_handler`)
Fora del router i del middleware de Firebase: s'autentica amb la capçalera `X-API-Key` (`MOBILICAT_API_KEY`).
**Excepció:** claus JSON en **snake_case**, segons el contracte acordat amb Mobilicat (DP-032).

| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /api/v1/congestion/estimate` | `CongestionHandler.Estimate` | Factor de congestió (0.1–1.0), nivell (`LOW`/`MEDIUM`/`HIGH`) i nombre d'esdeveniments per a 1–100 ubicacions (lat/lng WGS84 + `radius_km`, 1 km per defecte) en una franja (`start_time`/`end_time`, ara per defecte) | Implementat (DP-032) |

```json
// Petició
{ "timeframe": { "start_time": "2026-10-15T18:00:00Z", "end_time": "2026-10-15T22:00:00Z" },
  "locations": [ { "id": "punt_1", "latitude": 41.3871, "longitude": 2.1701, "radius_km": 1.5 } ] }
// Resposta 200
{ "results": [ { "id": "punt_1", "latitude": 41.3871, "longitude": 2.1701,
                 "congestion_factor": 0.85, "level": "HIGH", "events_count": 4 } ] }
```

---

## Endpoints planificats (Sprint 1 - En curs)

| Mètode i ruta | Història / Tasca | Descripció |
|---|---|---|
| `DELETE /events/{id}/enroll` | **US #49** (Tasca #186) | Desapuntar-se d'un esdeveniment (decrementar places i eliminar de l'agenda) |
| `GET /users/me` i `GET /users/{id}` | **US #36** (Tasca #165) | Obtenir el perfil propi o d'un altre usuari amb mètriques i plans |
| `PATCH /users/me` | **US #12** (Tasca #119) | Modificar nom, biografia i foto de perfil de l'usuari |
| `GET /preferences/test` | **US #27** (Tasca #226) | Obtenir els 25 esdeveniments representatius per al test d'interessos |
| `POST /preferences/test` | **US #27** (Tasca #226) | Enviar les respostes del test i calcular pesos per categoria |

---

## Serveis consumits (sempre des del backend)

| Servei | Ús | Si falla |
|---|---|---|
| **Supabase (PostgreSQL 15+)** | Base de dades principal gestionada al cloud amb Row Level Security (RLS) habilitat | L'API retorna error 500 |
| **Firebase Auth** | Validació de tokens JWT d'usuaris (`firebase_client`) | Petició no autoritzada (401) |
| **Dades Obertes de Catalunya** | Ingesta diària automàtica d'esdeveniments oficials (`gencat_client`) | Es manté l'últim catàleg sincronitzat |
| **Groq AI (Llama 3.3 70B)** | Processament de llenguatge natural i orquestració de consultes per al xatbot | Error 502/504 en el xatbot; la resta de l'app funciona normalment |
| **Servidor MCP (`cmd/mcp`)** | Eines de catàleg per a Groq AI mitjançant el protocol MCP | Error 502 en el xatbot |
| **Google Maps API** | Geocodificació i visualització de mapa interactiu | El llistat d'esdeveniments continua actiu |
| **Talaia (PES)** | Activitats astronòmiques externes | Mostra únicament activitats locals |
| **Mobilicat (PES)** | Ens consumeix: estimació de congestió (`POST /api/v1/congestion/estimate`) a partir dels esdeveniments i les inscripcions | Contracte provisional a DP-032; autenticació amb `X-API-Key` |

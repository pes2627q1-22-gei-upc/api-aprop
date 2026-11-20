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
| `GET /events/recommended` | `EventHandler.GetRecommendedEvents` | Llistat paginat d'esdeveniments actius recomanats amb puntuació i paginació a SQL (`recommendations.sql`) | Implementat (DP-036) |
| `GET /events/{id}` | `EventHandler.GetEvent` | Detall complet d'un esdeveniment per ID (inclou `creator-name` i `creator-reliability` per als no oficials, `null` als oficials o sense valoracions) | Implementat |
| `POST /events` | `EventHandler.CreateEvent` | Creació d'esdeveniments no oficials | Implementat |
| `PUT /events/{id}` | `EventHandler.UpdateEvent` | Modificació d'un esdeveniment no oficial: només el creador (`403` si no ho és o si és oficial), només si l'estat és `Active` (`409` si no, p. ex. `Withdrawn`) i complint R5 contra la data de creació (`400`) | Implementat (DP-038) |
| `DELETE /events/{id}` | `EventHandler.DeleteEvent` | Esborrat d'un esdeveniment no oficial i de les dades que en depenen (només el creador; 204) | Implementat |
| `POST /events/{id}/cancel` | `EventHandler.CancelEvent` | Cancel·lació d'un esdeveniment no oficial: només el creador (`403`). En una transacció, l'estat passa d'`Active` a `Cancelled` i cada inscrit (excepte el creador) rep una notificació `Cancelacio` («<títol> s'ha cancel·lat.»). `200` amb `{event-id, state, notified-users}`; `409` si ja no és actiu o ja ha acabat; `404` si no existeix. Després, `PUT /events/{id}` respon `409` | Implementat (Tasca #180, DP-041) |
| `GET /events/map` | `EventHandler.ListMapEvents` | Esdeveniments amb coordenades i metadades bàsiques per al mapa | Implementat |
| `GET /categories` | `EventHandler.GetCategories` | Llistat de totes les categories del catàleg | Implementat |
| `GET /categories/{id}` | `EventHandler.GetCategory` | Detall d'una categoria específica | Implementat |

### Inscripcions i Agenda (`signup_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /events/{id}/signup` | `SignUpHandler.SignUp` | Inscripció d'un usuari a un esdeveniment | Implementat |
| `GET /users/{userId}/events` | `SignUpHandler.ListUserEvents` | Consulta de les activitats a les quals s'ha apuntat l'usuari (Agenda) | Implementat |

### Valoracions (`rating_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /events/{id}/ratings` | `RatingHandler.Rate` | Valora un esdeveniment assistit (1–5; R9/R10) | Implementat (`feature/event-rating`, DP-033) |
| `PUT /events/{id}/ratings/me` | `RatingHandler.UpdateMine` | Modifica la meva valoració (cos `{"score":1..5}`; `200` amb la valoració; `400`/`401`; `404` si no existeix l'esdeveniment o la valoració) | Implementat (Tasca #205, DP-036) |
| `DELETE /events/{id}/ratings/me` | `RatingHandler.DeleteMine` | Elimina la meva valoració (`204`; `401`; `404` si no existeix l'esdeveniment o la valoració). `Users.Rating` (fiabilitat) es recalcula amb el trigger d'`EventRatings` | Implementat (Tasca #205, DP-036) |
| `GET /events/{id}/enrollment/attendance` | `RatingHandler.HasAttended` | Consulta si l'usuari autenticat ha confirmat l'assistència (`has-attended`, R10) | Implementat (`feature/event-rating`, DP-033) |

### Xat d'esdeveniment (`message_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `GET /events/{id}/messages?pageNum=<n>&pageSize=<n>` | `MessageHandler.ListMessages` | Missatges del xat de l'esdeveniment, del més nou al més antic, amb l'autor (`user-id`, `username`, `name`, `profile-picture`). Només per als inscrits (R11, `403`). Resposta `{event-id, total, page-num, page-size, messages: [...]}`; `pageNum` des de 0, `pageSize` 50 per defecte i màxim 100 (`400` si no); `404` si l'esdeveniment no existeix | Implementat (Tasca #147, DP-040) |
| `POST /events/{id}/messages` | `MessageHandler.SendMessage` | Envia un missatge (cos `{"text": "..."}`, retallat, 1–1000 caràcters). R11: l'autor ha d'estar inscrit (`403`). `201` amb el missatge; `400`/`401`/`404`; `409` si el mateix usuari n'envia dos en el mateix instant (clau R1). Taula `EventMessages` (migració 000019); el text es desa comprimit amb DEFLATE quan ocupa menys (`bytea` amb byte de format) | Implementat (Tasca #147, DP-040) |

### Assistent Virtual i Xatbot (`assistant_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /assistant/messages` | `AssistantHandler.SendMessage` | Envia un missatge a l'assistent; orquestra Groq AI i eines MCP per retornar resposta i recomanacions | Implementat |
| `GET /assistant/conversation` | `AssistantHandler.GetConversation` | Recupera l'historial de la conversa activa de l'usuari | Implementat |

### Autenticació i Usuaris (`auth_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `POST /auth/sync` | `AuthHandler.Sync` | Sincronitza l'usuari autenticat amb Firebase a la taula d'usuaris de Supabase | Implementat |
| `POST /auth/password-reset` | `PasswordResetHandler.Reset` | **Públic** (sense `Authorization`; fora del router). Cos `{"identifier": "<correu o nom d'usuari>"}`. Fa que Firebase enviï el correu de restabliment de contrasenya (en català). `204` tant si el compte existeix com si no; `400` identificador invàlid; `429` + `Retry-After` si es repeteix abans de 60 s; `502` si Firebase falla; `503` sense `FIREBASE_WEB_API_KEY` | Implementat (DP-039) |

### Compte d'usuari (`user_handler`)
| Mètode i ruta | Handler | Descripció | Estat |
|---|---|---|---|
| `GET /users?q=<query>&pageNum=<n>&pageSize=<n>` | `UserHandler.SearchUsers` | Cerca usuaris pel nom d'usuari o el nom (conté el text, sense distingir majúscules; els comodins `%` i `_` es cerquen literalment). Exclou l'usuari que cerca, els usuaris bloquejats en qualsevol sentit i els comptes que no estan en estat `Active`. Primer les coincidències exactes i les que comencen pel text. Paginat com `GET /events/search` (`pageNum` des de 0, `pageSize` 20 per defecte, màxim 50). Resposta `{query, total, page-num, page-size, users: [{id, username, name, profile-picture}]}`; `q` buit → pàgina buida; `400` paginació invàlida o `q` > 100 caràcters; `401` sense identitat | Implementat (DP-037) |
| `POST /users/{id}/block` | `UserHandler.BlockUser` | Bloqueja l'usuari (`{id}`: UUID, Firebase UID o nom d'usuari). En una transacció elimina l'amistat i passa a `Rejected` les sol·licituds pendents o acceptades entre tots dos (R3). `201` amb `{id, username, name, profile-picture, date-blocked}`; `400` si és un mateix (R2); `404` si no existeix; `409` si ja estava bloquejat. No es notifica el bloquejat | Implementat (Tasca #108, DP-042) |
| `DELETE /users/{id}/block` | `UserHandler.UnblockUser` | Desbloqueja l'usuari (`204`; `404` si no existeix o no estava bloquejat). L'amistat no es recupera | Implementat (Tasca #108, DP-042) |
| `GET /users/me/blocked` | `UserHandler.ListBlockedUsers` | Usuaris que he bloquejat, del més recent al més antic (pantalla «Usuaris bloquejats», fig. 19). Llista sense paginació | Implementat (Tasca #108, DP-042) |
| `DELETE /users/me` | `UserHandler.DeleteMyAccount` | Baixa del compte (RGPD): esborra en una transacció l'usuari, la seva configuració, els esdeveniments no oficials que ha creat (amb les dades que en depenen) i totes les files que el referencien; dins la transacció esborra el compte de Firebase Auth i, després del commit, l'avatar del bucket privat (best-effort). `204`; `401` sense identitat; `404` usuari inexistent; `502` si Firebase falla (no s'esborra res) | Implementat (DP-031) |
| `POST /users/me/avatar` | `UserHandler.UploadAvatar` | Puja la foto de perfil (multipart `avatar` o `file`; JPEG, PNG o WebP; màxim 5 MB) al bucket **privat** `avatars` de Supabase Storage i esborra l'anterior. `200` amb el perfil; `400` imatge invàlida; `502` si Storage falla; `503` sense `SUPABASE_URL`/`SUPABASE_SECRET_KEY` | Implementat (Tasca #247, DP-044) |

### Fotos de perfil privades (Tasca #247)
Les fotos pujades es desen al bucket privat `avatars` (`S3_AVATAR_BUCKET`) i `Users.ProfilePicture` guarda la ruta de l'objecte
(`<uuid>/avatar-<ms>.<ext>`), no una URL. Totes les respostes que inclouen `profile-picture` (`GET /users/{id}`, `PATCH /users/me`,
`POST /users/me/avatar`, `GET /users?q=`, `POST /users/{id}/block`, `GET /users/me/blocked`, `GET`/`POST /events/{id}/messages` i
`POST /auth/sync`) la retornen com una **URL signada** que caduca en 1 hora (la mateixa URL es reutilitza mentre li queden ≥ 30 min). Les
URL externes (p. ex. la foto de Google) es retornen tal qual. Si no es pot signar, `profile-picture` surt buit (o l'avatar generat al
perfil). `PATCH /users/me` només accepta `profile-picture` buit o una URL `http(s)` (`400` en cas contrari). Detalls a DP-044.

### Idioma i traducció d'esdeveniments (US #28)
Idiomes: `ca` (original, per defecte), `es`, `en`. L'idioma de l'usuari es guarda a `Settings.Language` i es llegeix/modifica
amb el camp `language` de `GET /users/me` (només al perfil propi) i `PATCH /users/me` (`400` si no és `ca`/`es`/`en`).

Els endpoints que retornen esdeveniments (`GET /events`, `/events/{id}`, `/events/search`, `/events/map`, `/event/map`, `/events/recommended`,
`/events/popular`, `/users/{id}/created-events`, `/users/{userId}/events`) accepten `?lang=ca|es|en`. Resolució: `?lang` → idioma de
l'usuari autenticat → original. `lang` invàlid → `400`. Es tradueixen `title`, `subtitle`, `description`, `schedule`, `tickets` i
`url-names` (cada nom d'enllaç separat per comes); `place`, `municipality`, `address`, etc. queden sense traduir.
El xatbot (`assistant_handler`) **no** tradueix (DP-043).

Traducció: `translate_client` (`POST {TRANSLATOR_URL}/translate`, `TRANSLATOR_API_KEY` opcional; desplegat sobre Cloudflare Workers AI) →
`translation_service`, que cacheja a `EventTranslations` (clau `(Event, Lang)`, `SourceHash` = SHA-256 dels sis camps
originals: si l'esdeveniment canvia, es torna a traduir). Sense `TRANSLATOR_URL`, o si el servei falla, es retorna el text original
(i es registra el motiu); mai no falla la petició. Detalls i supòsits a DP-043.

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
| **Supabase Storage** | Fotos de perfil al bucket privat `avatars` (`s3_client`, API REST amb `SUPABASE_SECRET_KEY`; URLs signades) | Pujada: `502`/`503`; lectura: `profile-picture` buit |
| **Firebase Auth** | Validació de tokens JWT d'usuaris (`firebase_client`) | Petició no autoritzada (401) |
| **Dades Obertes de Catalunya** | Ingesta diària automàtica d'esdeveniments oficials (`gencat_client`) | Es manté l'últim catàleg sincronitzat |
| **Groq AI (Llama 3.3 70B)** | Processament de llenguatge natural i orquestració de consultes per al xatbot | Error 502/504 en el xatbot; la resta de l'app funciona normalment |
| **Servidor MCP (`cmd/mcp`)** | Eines de catàleg per a Groq AI mitjançant el protocol MCP | Error 502 en el xatbot |
| **Google Maps API** | Geocodificació i visualització de mapa interactiu | El llistat d'esdeveniments continua actiu |
| **Talaia (PES)** | Activitats astronòmiques externes | Mostra únicament activitats locals |
| **Mobilicat (PES)** | Ens consumeix: estimació de congestió (`POST /api/v1/congestion/estimate`) a partir dels esdeveniments i les inscripcions | Contracte provisional a DP-032; autenticació amb `X-API-Key` |

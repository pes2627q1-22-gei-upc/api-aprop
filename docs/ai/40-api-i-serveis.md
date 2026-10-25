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
| **Mobilicat (PES)** | Consum de l'API d'esdeveniments i afluència per part d'altres equips de PES | Contracte d'integració a través dels endpoints REST públics |

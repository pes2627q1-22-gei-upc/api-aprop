# Decisions pendents

> **Quan llegir-ho:** abans de començar una tasca (per saber què no està decidit), quan facis un canvi sense
> estar-ne segur i en revisar una PR. Hi registren entrades tant els agents d'IA com les persones.
> Vigilància: el subagent `vigilant-decisions` (després d'implementar, abans de `/pr` i `/revisar`).

## Quan s'hi ha d'afegir una entrada

Sempre que es faci (o es deixi de fer) un canvi basat en un supòsit que ningú no ha confirmat:

- requisit o criteri d'acceptació ambigu o contradictori (Taiga, PDFs, mockups);
- contracte del backend o d'un altre equip que encara no existeix;
- dependència, paquet o servei nou;
- desviació del mockup, de l'arquitectura o d'una convenció;
- dades, valors o noms deduïts (enums, llindars, textos);
- dubte de seguretat, privacitat o integritat.

**Al codi**, el punt afectat es marca amb `TODO(DP-NNN)` (no amb un `TODO` genèric). A la resposta a l'usuari,
l'agent indica quines DP ha obert. Mai no es presenta un supòsit com a decisió presa.

## Format

```markdown
### DP-NNN · <Títol curt>
- **Estat:** oberta | resolta AAAA-MM-DD | descartada AAAA-MM-DD
- **Obert per:** AAAA-MM-DD · <persona o agent (eina, model)> · Taiga #<ID> si n'hi ha
- **Context:** <fitxers o pantalla afectats>
- **Dubte:** <què no se sap, en 1–2 línies>
- **Opcions:** A) … · B) …
- **Provisionalment:** <què s'ha fet mentrestant i on hi ha el `TODO(DP-NNN)`>, o «res, bloqueja la tasca»
- **Qui decideix:** equip | frontend | backend | PO (professor) | altre equip
- **Resolució:** <quan es tanqui: decisió + on queda (ADR, memoria.md, codi)>
```

Regles: numeració correlativa (no es reutilitza cap número); les noves van al final d'«Obertes». En resoldre una DP,
omple «Resolució», canvia'n l'estat, mou-la a «Resoltes» i substitueix els `TODO(DP-NNN)` del codi. Si la
resolució és d'arquitectura, crea un ADR; si és una convenció, afegeix-la a `docs/ai/memoria.md`.

## Obertes

### DP-001 · Internacionalització de la UI
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `lib/` (textos dins de les pantalles), skill `i18n-textos`
- **Dubte:** l'NFR demana textos en fitxers de traducció i ca/es/en; el mockup de Configuració només mostra ca/es; `AGENTS.md` prohibeix afegir paquets sense necessitat.
- **Opcions:** A) `flutter_localizations` + `intl` amb ARB (ca/es/en) · B) el mateix però només ca/es · C) ajornar-ho fins a la història «Canviar idioma»
- **Provisionalment:** UI només en català; cap paquet afegit.
- **Qui decideix:** equip
- **Resolució:**

### DP-002 · Llibreria de migracions de BD
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `api-aprop/migrations/`, skill `migracio-bd`
- **Dubte:** la memòria diu «llibreria migrations», però no és a `go.mod`.
- **Opcions:** A) `golang-migrate/migrate` amb `embed` · B) una altra llibreria · C) un runner propi mínim amb `database/sql`
- **Provisionalment:** convenció de fitxers `NNNNNN_nom.up.sql` / `.down.sql`.
- **Qui decideix:** backend
- **Resolució:**

### DP-003 · Mètode d'accés: Google o usuari i contrasenya
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `lib/features/auth/`, mockups (Incepció 2 §8), Taiga #7
- **Dubte:** els mockups preveuen l'accés amb Google; el codi i els criteris de #7 implementen usuari o correu + contrasenya.
- **Opcions:** A) tots dos via Firebase · B) només Google · C) només correu i contrasenya
- **Provisionalment:** comptes de demostració amb contrasenya; botó de Google desactivat.
- **Qui decideix:** equip / PO
- **Resolució:**

### DP-004 · Claus de Firebase i Google Maps al client
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `docs/ai/30-arquitectura.md`, ADR-0003
- **Dubte:** els PDFs diuen que totes les claus van al backend, però el login de Firebase i el SDK de Maps normalment s'executen al client amb claus restringides.
- **Opcions:** A) SDK al client amb claus restringides per app · B) tot via backend (proxy)
- **Provisionalment:** res integrat.
- **Qui decideix:** equip (cal un ADR)
- **Resolució:**

### DP-005 · Clau única dels esdeveniments oficials
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `event_model.Event.Code`, UML R1 «(Oficial, codi)»
- **Dubte:** el `Code` de la Generalitat no és únic (es reutilitza), però el UML el fa servir com a clau.
- **Opcions:** A) clau composta (codi + data d'inici + espai) · B) ID propi + `UNIQUE` a una altra combinació · C) corregir el UML
- **Provisionalment:** ID intern assignat pel repositori.
- **Qui decideix:** backend
- **Resolució:**

### DP-006 · Contractes amb Talaia i Mobilicat
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `docs/ai/40-api-i-serveis.md`
- **Dubte:** rutes, formats, autenticació i versionat dels serveis que intercanviem.
- **Opcions:** —
- **Provisionalment:** interfície + fake al backend quan es comenci la integració. Mobilicat: primer endpoint `POST /api/v1/congestion/estimate` implementat; supòsits a DP-032.
- **Qui decideix:** altre equip + backend
- **Resolució:**

### DP-007 · Contrast de `AppTheme.muted` sobre `cream`
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `lib/app/theme/app_theme.dart`
- **Dubte:** ≈ 4,4:1 sobre `cream`, per sota de l'AA de WCAG (4,5:1) en text normal; sobre `cardBackground` és ≈ 4,9:1.
- **Opcions:** A) enfosquir `muted` · B) fer-lo servir només sobre targetes o en text gran
- **Provisionalment:** cap canvi al tema.
- **Qui decideix:** frontend
- **Resolució:**

### DP-008 · Primera versió SemVer
- **Estat:** resolta 2026-10-28
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** skill `release-semver`
- **Dubte:** la primera release ha de ser `v0.1.0` o el MVP ha de ser `v1.0.0`?
- **Opcions:** A) `v0.x` fins al MVP i `v1.0.0` al MVP · B) `v1.0.0` al final de l'Sprint 1
- **Provisionalment:** res.
- **Qui decideix:** equip
- **Resolució:** es publica la versió `v1.0.0` com a primera release en tancar el Sprint 1 (MVP) fusionant a la branca `main`, alineant-se amb la release `v1.0.0` de `frontend-aprop`.

### DP-009 · Atributs i valors d'enum que només són al diagrama UML
- **Estat:** oberta
- **Obert per:** 2026-10-16 · configuració IA (Cursor)
- **Context:** `docs/ai/20-model-domini.md` (marques `TODO(confirmar)`)
- **Dubte:** el diagrama és una imatge; falten atributs i valors exactes (estat del compte, del NoOficial, de la denúncia, llindars d'XP).
- **Opcions:** —
- **Provisionalment:** noms deduïts de les restriccions textuals.
- **Qui decideix:** qui va fer el diagrama
- **Resolució:**

### DP-010 · Enrollment backend contract and Calendar integration
- **Estat:** oberta
- **Obert per:** 2026-10-18 · Codex · Taiga #82
- **Context:** `EnrollmentRepository`, `DemoEnrollmentRepository`, list/detail participation controls and future Agenda integration.
- **Dubte:** enrollment endpoints, error format, authenticated identity and persistence are not agreed in the available backend contract. The calendar feature is developed separately and is not present on `develop`.
- **Opcions:** A) implement an authenticated API adapter once backend and Calendar agree the contract · B) keep the frontend on the shared demo repository until then.
- **Provisionalment:** implement a Dart-only repository contract with confirmed, idempotent enrollment and a stream of enrolled activity IDs per user. Demo data is retained in memory across sign-outs and cleared on app shutdown. Calendar will subscribe to the same repository; no HTTP endpoints, enrollment DTOs or calendar screen are invented. `TODO(DP-010)` marks the demo adapter.
- **Qui decideix:** frontend + backend + calendar owner
- **Resolució:**

### DP-011 · Identitat provisional `X-User-ID` als endpoints d'inscripció
- **Estat:** oberta
- **Obert per:** 2026-10-19 · agent (Cursor, Claude) amb confirmació de l'usuari · Taiga #39
- **Context:** `api-aprop`: `POST /events/{id}/signup` (capçalera `X-User-ID`) i `GET /users/{userId}/events`
- **Dubte:** encara no hi ha Firebase Auth al backend ni taula d'usuaris; qualsevol client pot apuntar o consultar els plans d'un altre UUID.
- **Opcions:** A) validar el token de Firebase en un middleware i obtenir-ne l'usuari (eliminar `X-User-ID` i comprovar que `{userId}` és el de la sessió) · B) mantenir-ho només en desenvolupament
- **Provisionalment:** l'UUID arriba a la capçalera o a la ruta sense autorització; `TODO(DP-011)` als handlers. El frontend continua amb el repositori de demostració (DP-010).
- **Actualització 2026-10-26** (Claude Code, branca `fix/resolve-user-id`): inscripcions (`signup`/`enroll` i estat), xatbot i preferències apliquen l'opció A: amb el middleware, l'usuari surt del token de Firebase i es tradueix a UUID (`internal/handlers/requester`), i `X-User-ID` s'ignora; només es llegeix sense middleware. Queda oberta per a `GET /users/{userId}/events` (no es comprova que `{userId}` sigui el de la sessió), les notificacions (`/notifications/user/{id}`) i el `userId` del mapa.
- **Actualització 2026-11-11** (Claude Code, branca `feature/unificar-usuari`, Taiga #246): events i users també identifiquen el requester amb `internal/handlers/requester`, el resolver és únic (`auth_service.ResolveUserID`) i sense identitat sempre es respon `401`. Per decisió de l'usuari, les notificacions **no** s'han tocat: `GET /notifications/user/{id}` i `POST /notifications/user/{id}/read` continuen sense comprovar que `{id}` sigui el de la sessió (`TODO(DP-011)` al handler).
- **Qui decideix:** backend + equip (ADR-0003)
- **Resolució:**

### DP-012 · Unicitat de `SignUp` (R1) sense restricció a la BD
- **Estat:** oberta
- **Obert per:** 2026-10-19 · agent (Cursor, Claude) amb confirmació de l'usuari · Taiga #39
- **Context:** `api-aprop/migrations/000001_initial_migration.up.sql` (PK `(UserID, Event, SignUpDate)`), `queries/signups.sql`
- **Dubte:** la clau primària inclou la data, de manera que la BD admet inscripcions duplicades del mateix usuari al mateix esdeveniment.
- **Opcions:** A) migració amb `UNIQUE (UserID, Event)` (eliminant duplicats) + `ON CONFLICT DO NOTHING` · B) mantenir la comprovació a la consulta
- **Provisionalment:** `INSERT … WHERE NOT EXISTS`, idempotent en peticions seqüencials però no davant peticions concurrents; la llista fa `DISTINCT`. `TODO(DP-012)` a la consulta/repositori.
- **Qui decideix:** backend
- **Resolució:**

### DP-013 · Dates dels plans a l'Agenda: diversos dies, permanents, sense data i hora
- **Estat:** oberta
- **Obert per:** 2026-10-19 · vigilant-decisions (Cursor) · Taiga #39 / #84
- **Context:** `AgendaViewModel._resolvePlans` (agrupa per `_dateOnly(activity.startsAt)`), `AgendaPlanCard` (`formatAgendaTime(startsAt)`), `Activity` (només `startsAt`); `api-aprop`: `Events.StartDate` és `Date` i admet `NULL` (migració 000002), `EndDate`, `Permanent`, `Schedule` en text lliure, i `ListUserSignedUpEvents` filtra per solapament `[StartDate, COALESCE(EndDate, StartDate)]`.
- **Dubte:** el frontend posa el punt només al dia d'inici, mentre que el backend ja retorna els esdeveniments de diversos dies per solapament; tampoc no hi ha criteri per als permanents o sense data d'inici, ni d'on surt l'hora «19.00 h» si el backend només té dates i un horari en text.
- **Opcions:** A) afegir `endsAt` (i una hora opcional) a `Activity` i marcar tots els dies de l'interval · B) marcar només el dia d'inici i mostrar el període a la targeta · C) excloure de l'Agenda els permanents/sense data
- **Provisionalment:** només `startsAt` (documentat com a fora d'abast a `docs/calendar.md`); sense marcador al codi.
- **Qui decideix:** frontend + backend (+ PO per als permanents)
- **Resolució:**

### DP-014 · Desviacions de l'Agenda respecte del mockup (fig. 11) i textos inventats
- **Estat:** oberta
- **Obert per:** 2026-10-19 · vigilant-decisions (Cursor) · Taiga #39 / #84
- **Context:** `agenda_plan_card.dart`, `agenda_date_formatter.dart` (`formatAgendaMonth`), `agenda_screen.dart`, `main_shell.dart`; `docs/source/incepcio-2.pdf` p. 25
- **Dubte:** el mockup i el seu text diuen que la targeta mostra categoria (xip), municipi, hora, **procedència** («Font: Agenda Cultural · Generalitat») i l'etiqueta «Apuntat ✓»; la implementació no mostra ni la procedència ni «Apuntat» i posa hora i categoria en una sola línia. El títol del mes és «Setembre de 2026» (mockup: «Setembre 2026») i s'ha eliminat el subtítol «Els teus plans culturals». Els textos del dia buit («No tens cap pla per a aquest dia.», «Apunta’t a activitats des d’Inici…») i d'error no són al mockup (només diu «missatge específic»).
- **Opcions:** A) alinear la targeta, el títol i el subtítol amb el mockup i validar els textos · B) acceptar la versió compacta i actualitzar `docs/ai/50-ui-mockups.md`
- **Provisionalment:** 2026-10-19, confirmat per l'usuari: la targeta ja mostra la procedència («Font: Agenda Cultural · Generalitat» o «Publicat per …») i «Apuntat ✓», i el títol es manté «Setembre de 2026» (forma catalana correcta). Queden per validar amb l'equip el subtítol i els textos del dia buit i d'error; sense marcador al codi.
- **Qui decideix:** frontend + equip (disseny)
- **Resolució:**

### DP-015 · Accessibilitat del calendari mensual
- **Estat:** oberta
- **Obert per:** 2026-10-19 · vigilant-decisions (Cursor) · Taiga #39 / #84
- **Context:** `month_calendar.dart` (`_DayCell`, punt `colors.secondary`)
- **Dubte:** el punt coral sobre `cream` té un contrast ≈ 2,7:1 (WCAG 1.4.11 en demana 3:1 per a elements gràfics; la semàntica sí que diu «amb plans»). Les cel·les fan 48 px d'alt, però l'amplada és 1/7 de la pantalla: a 320 px queden ≈ 41 px, per sota dels 48 dp recomanats.
- **Opcions:** A) enfosquir el punt o fer-lo més gran i reduir el farciment lateral per assolir 48 dp · B) acceptar-ho (el color ve del mockup) i documentar-ho
- **Provisionalment:** 2026-10-19, confirmat per l'usuari: el punt passa a `forest` (`colorScheme.primary`, ≈ 12:1 sobre `cream`; `cream` sobre el dia seleccionat). Queda oberta l'amplada de les cel·les en pantalles estretes; sense marcador al codi.
- **Qui decideix:** frontend
- **Resolució:**

### DP-016 · Contracte d'API de l'assistent (xatbot) amb el backend
- **Estat:** oberta
- **Obert per:** 2026-10-19 · Antigravity · Taiga #91 / #14
- **Context:** `AssistantRepository`, `DemoAssistantRepository`, `AssistantMessage`, `AssistantInterpretation` (`lib/features/assistant/`).
- **Dubte:** l'endpoint HTTP del backend per consultar Gemini, el format JSON de la consulta/resposta (criteris interpretats de tipus, municipi i dates, i llista d'activitats recomanades) i la persistència de `ConversaXatbot` no estan definits a `api-aprop`.
- **Opcions:** A) l'endpoint retorna el text, els filtres interpretats i les activitats completes · B) l'endpoint retorna IDs d'activitats que es resolen contra `ActivityRepository`.
- **Provisionalment:** contracte `AssistantRepository` en Dart pur i `DemoAssistantRepository` (marcat amb `TODO(DP-016)`) que interpreta tipus, municipi i dates en local sobre `ActivityRepository` sense inventar rutes ni DTO HTTP.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-017 · Contracte del backend per crear activitats no oficials
- **Estat:** oberta
- **Obert per:** 2026-10-19 · Claude Code (Opus 5.5) · Taiga #93 / #31
- **Context:** `ActivityRepository.createActivity`, `NewActivity`, `DemoActivityRepository`, `MainShell._showCreatedActivity`.
- **Dubte:** no hi ha endpoint per publicar un `NoOficial` a `api-aprop`: ruta, JSON, format del preu (s'ha fet servir `double` en euros, `0` = gratuïta) i si el backend apunta automàticament la persona creadora.
- **Opcions:** A) el backend crea l'activitat i la inscripció de qui la crea en una sola operació · B) el frontend crida després l'endpoint d'inscripció.
- **Provisionalment:** `createActivity` en Dart pur i implementació en memòria; `MainShell` apunta la creadora amb `EnrollmentViewModel` (`TODO(DP-017)`).
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-018 · Categories de les activitats: llista i nombre per activitat
- **Estat:** oberta
- **Obert per:** 2026-10-19 · Claude Code (Opus 5.5) · Taiga #93 / #31
- **Context:** `activity_options.dart` (`activityCategories`), `CreateActivityViewModel`, formulari «Crea una activitat».
- **Dubte:** la llista tancada de categories no està definida, i el CA1 diu «categoria (≥ 1)» mentre que el model `Activity` en té una sola.
- **Opcions:** A) una sola categoria (com ara) · B) diverses categories (cal canviar `Activity.category` i la llista, la fitxa i l'agenda).
- **Provisionalment:** una sola categoria triada d'una llista provisional (Música, Teatre, Exposicions) (`TODO(DP-018)`). L'usuari ho ha decidit així de moment per no canviar el model.
- **Qui decideix:** equip
- **Resolució:**

### DP-019 · Imatge de portada de les activitats creades
- **Estat:** oberta
- **Obert per:** 2026-10-19 · Claude Code (Opus 5.5) · Taiga #93 / #94
- **Context:** `activity_options.dart` (`activityImageTemplates`), portada i panell «Imatge de portada» del formulari.
- **Dubte:** el CA1 diu que el cartell és opcional i es puja a S3; les plantilles definitives no existeixen i la pujada (#94) necessita un paquet (com `image_picker`) i un endpoint.
- **Opcions:** A) imatge obligatòria, amb una plantilla per defecte (com ara) · B) imatge opcional amb un marcador quan no n'hi ha.
- **Provisionalment:** les tres fotografies de demostració fan de plantilla; la primera és la portada per defecte; «Puja una foto» és al panell però desactivat (`TODO(DP-019)`).
- **Qui decideix:** equip
- **Resolució:**

### DP-020 · Ubicació exacta de les activitats creades i paquet de mapes
- **Estat:** oberta
- **Obert per:** 2026-10-19 · Claude Code (Opus 5.5) · Taiga #93 / #31 / #21
- **Context:** camp «Municipi» del formulari; història #21 (mapa). Relacionada amb DP-004.
- **Dubte:** el CA1 i el CA2 demanen espai/coordenades i que l'activitat surti al mapa; no hi ha paquet de mapes ni servei per passar d'adreça a coordenades.
- **Opcions:** A) `flutter_map` + OpenStreetMap (sense clau) · B) Google Maps (clau d'API, vegeu DP-004); en tots dos casos, centrar el mapa al municipi i deixar ajustar el marcador.
- **Provisionalment:** 2026-10-20, l'usuari (Sukhjeet) ha triat B: `google_maps_flutter` al formulari, amb la clau a `android/local.properties` (`docs/google-maps.md`); la clau de l'equip la crea un altre membre i iOS encara no està configurat. El municipi es converteix en coordenades amb una llista de demostració (`DemoLocationRepository`) i l'usuari ajusta el marcador (`TODO(DP-020)`). Cal que l'equip ho confirmi, perquè la història #21 farà servir el mateix paquet.
- **Qui decideix:** equip
- **Resolució:**

### DP-021 · Disseny del formulari de creació sense mockup
- **Estat:** oberta
- **Obert per:** 2026-10-19 · Claude Code (Opus 5.5) · Taiga #93
- **Context:** `CreateActivityScreen`, `start_date_time_sheet.dart`, botó flotant «Crear activitat» (`AppTheme.sage`).
- **Dubte:** el CA1 cita el botó de la fig. 7, però no hi ha mockup del formulari. L'usuari l'ha dissenyat prenent Partiful i Luma com a referència: portada a dalt, categories en fila horitzontal, rodes de data i hora, camp d'indicacions.
- **Opcions:** A) validar-lo tal com està · B) fer-ne un mockup i ajustar-lo.
- **Provisionalment:** implementat així (`TODO(DP-021)`).
- **Qui decideix:** equip
- **Resolució:**

### DP-022 · Mapa d'Inici: vista inicial, càrrega per zona i estil dels grups
- **Estat:** oberta
- **Obert per:** 2026-10-20 · Claude Code (Opus 5.5) · Taiga #21 / #79
- **Context:** `ActivitiesScreen` (selector Mapa / Llista), `activities_map.dart`, `DemoActivityRepository`.
- **Dubte:** el mockup (fig. 7) obre Inici amb el mapa; el CA3 demana carregar per zona visible en < 3 s, i no hi ha endpoint per zona; el número dels grups surt amb l'estil per defecte de Google, no amb el verd del mockup.
- **Opcions:** A) vista inicial mapa · B) vista inicial llista (com ara); per a la zona: endpoint amb els límits del mapa o carregar-ho tot; per als grups: acceptar l'estil de Google o fer l'agrupació a Dart amb icones pròpies.
- **Provisionalment:** la llista és la vista inicial (si Google Maps falla o no hi ha clau, Inici funciona igual); es mostren totes les activitats carregades; grups amb l'estil de Google (`TODO(DP-022)`).
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-023 · API d'Avisos, respostes d'amistat i accés als perfils
- **Estat:** oberta
- **Obert per:** 2026-10-19 · Codex · Taiga #86
- **Identificador:** renumerat de DP-017 a DP-023 el 2026-10-21 en integrar `develop`, per conservar la DP-017 de creació d’activitats sense col·lisions.
- **Context:** `features/notifications/`, mockups aprovats d'Avisos i Incepció 2 pp. 29–30; no hi ha contracte de notificacions/amistats ni funcionalitat de perfils disponible al client.
- **Dubte:** falten endpoints autenticats de bústia/lectura/resposta, persistència, sincronització d'amistats i enviament d'avisos. Les publicacions d'amics dels mockups tampoc no tenen una variant explícita al UML. Els criteris complets de Taiga #86 no són al repositori.
- **Opcions:** A) acordar contractes i integrar perfils/amistats · B) validar primer la pantalla i les interaccions amb dades demo aïllades.
- **Provisionalment:** projecció de bústia en Dart pur, dades demo per compte, lectura/resposta confirmades i callbacks de destinació. El perfil s'indica com a pendent fins que hi hagi un callback real. La resposta d'amistat simula només el resultat a la bústia: no implementa un graf d'amistats, push, programació de recordatoris, tipus UML nous ni rutes d'API. Els recordatoris històrics no creen inscripcions. Filtres desplaçables amb pantalla estreta/text ampliat, paletes centrals al tema i inicials si no hi ha avatar. `TODO(DP-023)` marca els límits demo/integració. Textos nous de càrrega/error/buit provisionals en català.
- **Qui decideix:** frontend + backend + responsables de perfils/amistats
- **Resolució:**

### DP-024 · Desplegament a EC2 des del CD
- **Estat:** oberta
- **Obert per:** 2026-10-25 · Claude Code (Opus 5.5) · Taiga US #222
- **Context:** `.github/workflows/go.yml`, job `publish`; `docs/ai/60-metodologia-i-qualitat.md` (release → imatge → GHCR → EC2).
- **Dubte:** no hi ha cap servidor EC2 ni secrets de desplegament (host, usuari, clau SSH, ruta del `docker-compose.yml`/`.env`) definits.
- **Opcions:** A) job de deploy per SSH que fa `docker compose pull && up -d` a l'EC2 · B) l'EC2 consulta GHCR per si mateix (p. ex. Watchtower).
- **Provisionalment:** el CD només publica la imatge a GHCR (`git tag` i `latest`) quan es fa push d'un tag. `TODO(DP-024)` al workflow.
- **Qui decideix:** backend (infraestructura)
- **Resolució:**

### DP-025 · Llindar de cobertura del 80 % al CI
- **Estat:** oberta
- **Obert per:** 2026-10-25 · Claude Code (Opus 5.5) · Taiga US #222
- **Context:** `.github/workflows/go.yml`, job `test`; `docs/ai/agents/devops-ci.md` demana un llindar del 80 %.
- **Dubte:** la cobertura total actual és del 36,4 % (inclou el codi generat per `sqlc` i paquets sense proves); si s'aplica el llindar ara, cap PR no es podria integrar.
- **Opcions:** A) aplicar el 80 % excloent `sqlc/`, `cmd/` i models · B) pujar-lo gradualment · C) només informar-ne.
- **Provisionalment:** el CI calcula la cobertura, la mostra al log i la desa com a artefacte, però no falla per cobertura.
- **Qui decideix:** equip
- **Resolució:**

### DP-026 · Imatge d'un esdeveniment en modificar-lo amb `PUT /events/{id}`
- **Estat:** oberta
- **Obert per:** 2026-10-27 · Claude Code (Opus 5.5) · Taiga Tasca #236
- **Context:** `EventService.UpdateUnofficialEvent`; la imatge es puja a part amb `POST /events/{id}/image`.
- **Dubte:** el `PUT` substitueix tots els camps i, si el cos no porta `images`, esborrava la imatge pujada. No hi ha cap contracte que digui si el formulari d'edició l'ha d'enviar.
- **Opcions:** A) un `images` buit conserva la imatge actual (no es pot treure) · B) el client sempre envia `images` i un buit l'esborra · C) endpoint `DELETE /events/{id}/image`.
- **Provisionalment:** A (`TODO(DP-026)`).
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-027 · Reimplementació de l'assistent: tool calling directe i cerca amb rànquing
- **Estat:** oberta
- **Obert per:** 2026-11-07 · Claude Code (Sonnet 5.5) · branca `feature/assistant-tool-calling`
- **Context:** s'elimina MCP (`cmd/mcp`, `internal/mcp_server`, `internal/clients/mcp_client`, `/mcp`, `MCP_TOKEN`); el servei crida les eines directament (`assistant_service/tools.go`). `SearchEvents` ignora accents (`unaccent`, migració 000012) i ordena per rellevància del text + pes màxim de `UserPreferences`.
- **Dubte:** (1) l'extensió `unaccent` i la consulta no s'han provat contra Supabase; (2) els pesos de rellevància (títol 3, categoria 2, subtítol 1, descripció 1) i la suma amb la preferència són deduïts, no mesurats; (3) `ADR-0005` i `docs/architecture.md` encara descriuen MCP; (4) falten les eines `list_categories`/`get_user_context`, el filtre de tema fusionat i els IDs recomanats a l'historial; (5) ajuda de l'app: `assistant_service/help_guide.md` s'ha extret del codi del frontend el 2026-11-07 i s'ha de mantenir sincronitzada; l'assistent respon en l'idioma de l'usuari tot i que la UI és només en català; (6) Gemini com a còpia de seguretat (`GEMINI_API_KEY`, `GEMINI_MODEL`) sense provar; (7) `GET /assistant/suggestions` no és a `api/openapi.yaml` ni el frontend l'usa; (8) la navegació des del xat requereix un contracte nou amb el frontend i no s'ha fet.
- **Opcions:** A) mantenir el rànquing actual · B) afegir cerca per text complet (`tsvector`) · C) afegir `pgvector` si A/B no basten.
- **Provisionalment:** A.
- **Qui decideix:** equip
- **Resolució:**


### DP-028 · Millores de l'assistent: accions de pantalla, feedback i límit de missatges
- **Estat:** oberta
- **Obert per:** 2026-11-07 · Claude Code (Sonnet 5.5) · branca `feature/assistant-tool-calling`
- **Context:** `POST /assistant/feedback` (migració 000013, taula `ChatbotFeedback`), camp `action.screen` a la resposta de `POST /assistant/messages`, límit de 20 missatges/minut per usuari (`429`), registre de les crides a eines, ajuda amb l'estat de l'usuari (`user-has-done-test`).
- **Dubte:** (1) `action.screen` és un enum tancat (`home`, `agenda`, `discover`, `notifications`, `profile`, `settings`, `interest-test`) deduït del tema de la guia (`topicScreens` a `help.go`); el frontend ha de dir quines pantalles existeixen de debò; temes sense pantalla clara (mapa, crear activitat, amics) no en porten; (2) `action` no es desa: una conversa recarregada amb `GET /assistant/conversation` no el porta; (3) el límit és en memòria (es perd en reiniciar i no es comparteix entre instàncies) i els 20/min no s'han validat amb ús real; (4) la migració 000013 no s'ha aplicat a Supabase ni es reflecteix a `supabase/migrations/`; la consulta de `sqlc` s'ha escrit a mà (sqlc no és instal·lat): cal `sqlc generate` i comparar; (5) `gemini-2.5-flash` retorna 404; el valor per defecte és ara `gemini-3.8-flash` (el que suggereix Google) però la crida de prova ha donat 503/timeout, així que Gemini continua sense verificar; (6) el model respon sovint en català a preguntes en castellà o anglès tot i la regla del prompt; (7) `api/openapi.yaml` no inclou els endpoints nous.
- **Opcions:** A) deixar-ho així i validar amb el frontend · B) desar `action` en una columna nova · C) límit compartit (Redis/BD).
- **Provisionalment:** A.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-029 · Confirmar l'assistència: codi d'error R8, camp i esquema
- **Estat:** oberta
- **Obert per:** 2026-11-14 · Claude Code (Sonnet 5.5) · branca `feature/attendance-confirmation` (Taiga #211, US #64)
- **Context:** `PATCH /events/{id}/enrollment/attendance` amb cos `{"attendance": true|false|null}` (el camp ha de ser present). Migració 000015: `SignUp.Attendance boolean` nul·lable (`null` = desconeguda). El codi sqlc afegit coincideix amb la sortida de `sqlc generate` v1.31.1 (verificat en una còpia; la resta de diferències de `sqlc` són prèvies).
- **Dubte:** (1) R8 viola → `409 Conflict` (alternativa `422`); (2) esborrar el valor (`null`) es permet abans de l'inici; (3) no inscrit → `404`; (4) el nom del camp i la forma de la resposta no tenen contracte amb el frontend; (5) la migració 000015 s'ha provat en un PostgreSQL 16 local (migracions 1–15 + SQL i servei reals), però no s'ha aplicat a Supabase; (6) es compara `StartDate < ara` amb l'hora del servidor, sense zona horària d'esdeveniment; com que `Events.StartDate` és de tipus `date` (sense hora), un esdeveniment d'avui ja compta com a començat des de les 00:00.
- **Opcions:** A) 409 · B) 422.
- **Provisionalment:** A.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-030 · Especificació de la consulta GetUserProfile quan es completin els Settings
- **Estat:** oberta
- **Obert per:** 2026-11-17 · Lluc Paez · branca `fix/db-corrections` (Storyless)
- **Context:** `GET /users/{id}`.
- **Dubte:** (1) La consulta GET /users/{id} actualment retorna el camp showPlans dels Settings de l'usuari, cosa que s'entén perquè és necessari per calcular si els plans de l'usuari objectiu s'han de mostrar a l'usuari que fa la petició. El problema és la resta dels camps, com ara NotificationBefore, però també totes les altres configuracions que s'hauran d'afegir més tard. Hem de decidir què retorna aquesta consulta i què no. Penso que tindria més sentit fer una consulta separada per la resta de configuracions, ja que és informació que no cal per mostrar el perfil d'un usuari (i que els altres usuaris no haurien de tenir).
- **Opcions:** A) GetUserProfile retorna ShowPlans però cap altre Setting. Els Settings s'obtindran amb una consulta separada GET users/{id}/settings o GET settings/{id} segons convingui. · B) GetUserProfile retorna tots els Settings.
- **Provisionalment:** A.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-031 · Baixa del compte (`DELETE /users/me`): abast de l'esborrat RGPD
- **Estat:** oberta
- **Obert per:** 2026-11-16 · Claude Code (Opus 5.5) · branca `feature/delete-users` (Taiga #116)
- **Context:** `UserRepository.DeleteUser` (`queries/account_deletion.sql`), `user_service.DeleteAccount`, `firebase_client.DeleteUser`, `s3_client.DeleteAvatar`. Opcions 1–3 confirmades per l'equip (Arnau) abans d'implementar.
- **Dubte / supòsits:**
  1. **Esborrat físic** (no anonimització): es treu la fila de `Users`, els seus `Settings` i totes les files que la referencien (notificacions pròpies i les d'altres sobre l'usuari, sol·licituds d'amistat, amistats, bloquejos, inscripcions, valoracions donades i **rebudes**, missatges, test i preferències, nivell, estat i converses del xatbot).
  2. Els **esdeveniments no oficials creats per l'usuari s'esborren** amb les dades que en depenen (inscripcions i valoracions d'altres usuaris incloses), reutilitzant la lògica de `DELETE /events/{id}` (`deleteEventTx`). Els altres usuaris no en reben cap avís.
  3. L'**avatar** s'esborra de S3 després del commit, en mode best-effort (un error només es registra). Les imatges dels esdeveniments esborrats **no** s'eliminen de S3. La petició `DELETE` a S3 no va signada, igual que la pujada actual.
  4. Ordre: transacció a la BD → baixa a Firebase (dins la transacció, abans del commit; si falla → `502` i rollback) → commit. Si el commit fallés després d'esborrar el compte de Firebase, les dades quedarien a la BD sense compte: es registra amb `slog.Error` i s'han d'esborrar a mà. Un compte de Firebase inexistent no és error (idempotent).
  5. Sense credencials de Firebase (desenvolupament local) només s'esborra de la BD.
  6. No es demana cap confirmació addicional (contrasenya o reautenticació recent) al backend: la confirmació és responsabilitat del frontend.
  7. No s'ha provat contra PostgreSQL/Supabase (sense BD a l'entorn de desenvolupament): només tests unitaris.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** equip (backend + frontend)
- **Resolució:**

### DP-032 · Contracte amb Mobilicat: estimació de congestió
- **Estat:** oberta
- **Obert per:** 2026-11-16 · Claude Code (Opus 5.5) · branca `feature/mobilicat-endpoints` (Taiga #112)
- **Context:** `POST /api/v1/congestion/estimate` (`congestion_handler`, `congestion_service`, `queries/congestion.sql`), segons el correu de Mobilicat. Opcions 1–4 confirmades per l'equip (Arnau) abans d'implementar; la resta són supòsits pendents de Mobilicat.
- **Dubte / supòsits:**
  1. JSON en **snake_case** (`start_time`, `radius_km`, `congestion_factor`, `events_count`) tal com el correu, excepció a la regla kebab-case del projecte.
  2. Autenticació amb capçalera `X-API-Key` comparada amb `MOBILICAT_API_KEY` (sense la clau configurada → `503`; clau absent o incorrecta → `401`). Cal fer arribar la clau a Mobilicat per un canal privat.
  3. Fórmula: `score = Σ(1 + inscripcions no marcades com a absents)`, `factor = 0.1 + 0.9·(1 − e^(−score/K))`, K = 20 (`MOBILICAT_CONGESTION_K`), arrodonit a 2 decimals. Nivells: `LOW` < 0.4 ≤ `MEDIUM` < 0.7 ≤ `HIGH`.
  4. Compten els esdeveniments (oficials i no oficials) amb coordenades dins del radi (haversine) i dates que se solapen amb la franja, excepte els de modalitat `Online`. Un esdeveniment no permanent sense data de fi dura un dia; un de permanent sense data de fi, indefinidament. Sense `StartDate` no compta.
  5. Granularitat de dia: `Events.StartDate/EndDate` són `date`, així que la franja es converteix a dies de `Europe/Madrid`; les hores de `start_time`/`end_time` no afinen més.
  6. Valors per defecte: sense `timeframe` → ara; només `start_time` → `end_time = start_time`; només `end_time` → `start_time = ara`; `end_time < start_time` → `400`. Radi per defecte 1 km (també si és 0), màxim 50 km.
  7. Límits: 1–100 ubicacions per petició; `id` obligatori i retornat; resultats en el mateix ordre. A més de `locations`, s'accepta un objecte `location` individual ("ubicació individual o array" del correu).
  8. Sense versionat més enllà del prefix `/api/v1` del correu ni límit de peticions (rate limit).
  9. Mètode `POST` tot i ser una operació de només lectura (segura i idempotent): el cos porta fins a 100 ubicacions amb `id` i una franja niuada, que no caben bé a la query string, i un `GET` amb cos no té semàntica definida (RFC 9110) i molts clients i proxies el descarten. Si Mobilicat ho demana, es pot afegir `GET /api/v1/congestion/estimate?latitude=…&longitude=…` per a una sola ubicació.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** Mobilicat + backend
- **Resolució:**

### DP-033 · Valorar un esdeveniment: taula reutilitzada, codi d'error R10 i consulta d'assistència
- **Estat:** oberta
- **Obert per:** 2026-11-17 · Claude Code (Sonnet 5.5) · branca `feature/event-rating` (Taiga #184, US #48)
- **Context:** `POST /events/{id}/ratings` amb cos `{"score": 1..5}` → `201 {"user-id","event-id","score"}` i `GET /events/{id}/enrollment/attendance` (alias `/signup/attendance`) → `200 {"user-id","event-id","has-attended"}`. La tasca demanava una taula `ratings`, però `EventRatings` ja existeix (migració 000001: PK `(UserID, Event)` = únic per usuari/esdeveniment, `CHECK rating 1..5` des de la 000008, FK a `Users` des de la 000009), així que es reutilitza i **no s'afegeix cap migració**. `ratings.sql.go` s'ha generat amb `sqlc` v1.31.1 a partir de `ratings.sql`.
- **Dubte:** (1) R10 (no ha assistit) → `403` (alternatives `409`/`422`); (2) valoració repetida → `409` (no s'actualitza; potser s'hauria de permetre editar); (3) `score` fora de 1..5 o no enter → `400`; (4) R8 no es torna a comprovar: R10 exigeix `attendance = true`, que R8 ja només deixa fixar un cop l'esdeveniment ha començat; (5) `RatingDate` s'omple amb `now()` de la BD; (6) `GET /events/{id}/enrollment/attendance` retorna `has-attended: true|false` per saber si mostrar el botó de valorar; (7) sense migració ni prova contra una BD real.
- **Opcions:** A) 403 + 409 · B) 422 + 409 · C) 403 + upsert (editar).
- **Provisionalment:** A.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-034 · Fórmula de recomanació en SQL, exclusió d'esdeveniments inscrits i paginació obligatòria
- **Estat:** oberta
- **Obert per:** 2026-11-17 · Antigravity · Taiga #51 (`feature/recommended-events`)
- **Context:** `internal/services/event_service/event_service.go` (`GetRecommendedEvents`), `internal/handlers/event_handler/event_handler.go` (`GetRecommendedEvents`) i `internal/repositories/postgres/queries/recommendations.sql` (`ListRecommendedEvents`)
- **Dubte:** (1) La tasca demana combinar `Interès./pes` (`UserPreferences`) i `SignUp`, però no fixa una fórmula exacta ni si els esdeveniments on l'usuari ja està inscrit s'han d'excloure o només penalitzar; (2) tampoc no especifica si `pageSize` i `pageNum` han de ser obligatoris amb `pageSize > 0` i `pageNum >= 0` (`400` en cas contrari) o si han de tenir valors per defecte com a `GET /events` (`defaultPageSize = 20`, `pageNum = 0`); (3) `ListRecommendedEvents` a `recommendations.sql` calcula la puntuació, ordena i pagina (`LIMIT`/`OFFSET`) directament a PostgreSQL (generat amb `sqlc` v1.31.1), però encara no s'ha executat contra una BD PostgreSQL/Supabase real.
- **Opcions:** A) Excloure els esdeveniments on l'usuari ja està inscrit, puntuar a SQL com a mitjana per categoria de `(COALESCE(pes_preferència, 1.0) + LEAST(inscripcions_categoria * 0.5, 2.0))` + `LEAST(0.1 * LN(1 + inscripcions_globals), 0.5)` i exigir `pageSize > 0` i `pageNum >= 0` · B) Incloure els ja inscrits (penalitzats o no), ajustar pesos/topalls o posar valors per defecte de paginació.
- **Provisionalment:** A (`TODO(DP-034)` a `event_service.go` i `event_handler.go`).
- **Qui decideix:** backend + equip

### DP-035 · Fiabilitat de l'organitzador: font de les valoracions i llistat
- **Estat:** oberta
- **Obert per:** 2026-11-17 · Claude Code (Sonnet 5.5) · branca `feature/organizer-reliability` (US #59)
- **Context:** `reliability` = mitjana de `EventRatings.Rating` dels `UnofficialEvents` creats per l'usuari (`20-model-domini.md`: /fiabilitat); `null` si no n'hi ha. Per revisió de la PR #67 es llegeix de `Users.Rating`, que un trigger sobre `EventRatings` (migració 000017, amb backfill) recalcula a cada alta, edició o baixa de valoració. Nou `GET /users/{id}/ratings` (`id` o `me`) amb `event-id`, `event-title`, `rating`, `rating-date`.
- **Dubte:** (1) la revisió demanava calcular-ho des de `UserRatings`, però aquesta taula no té columna de puntuació (només rater, rated, event, ratingdate), així que la font continua sent `EventRatings`; (1b) el trigger és `SECURITY DEFINER` i no s'ha provat contra Supabase; (2) el llistat no exposa qui valora (privacitat) i és visible per a qualsevol usuari autenticat; (3) sense paginació; (4) la consulta no s'ha provat contra Supabase; `sqlc generate` v1.31.1 s'ha executat i només s'ha conservat `users.sql.go`.
- **Opcions:** A) EventRatings · B) UserRatings · C) mitjana de totes dues.
- **Provisionalment:** A.
- **Qui decideix:** equip

### DP-036 · Editar i eliminar la meva valoració (`/events/{id}/ratings/me`)
- **Estat:** oberta
- **Obert per:** 2026-11-18 · Claude Code (Sonnet 5.5) · branca `feature/rating-edit-delete-205` (Taiga #205, US #61)
- **Context:** `PUT /events/{id}/ratings/me` (cos `{"score": 1..5}`, el mateix que `POST`) → `200 {"user-id","event-id","score"}`; `DELETE /events/{id}/ratings/me` → `204`. L'usuari es resol de l'autenticació (només toca la seva valoració). Sense migració: `UPDATE`/`DELETE` sobre `EventRatings`. La fiabilitat de l'organitzador (`GetUserReliability`) és un `AVG` calculat en llegir, així que no cal recalcular res.
- **Dubte:** (1) `PUT` retorna `200` (el `POST` retorna `201`); (2) `404` tant si l'esdeveniment no existeix com si l'usuari no té valoració (mateix codi, missatge diferent); (3) `RatingDate` es conserva en editar (alternativa: posar-la a `now()`); (4) no es torna a comprovar R10 en editar/eliminar (la valoració ja existeix); (5) si s'ha de poder eliminar/editar una valoració d'un esdeveniment ja finalitzat o esborrat; (6) SQL no provat contra BD real.
- **Opcions:** A) `RatingDate` conservada · B) `RatingDate = now()` en editar.
- **Provisionalment:** A.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-037 · Cerca d'usuaris `GET /users?q=`
- **Estat:** oberta
- **Obert per:** 2026-11-18 · Claude Code (Opus 5.5) · branca `feature/cercar-usuaris` (Taiga #161)
- **Context:** `UserHandler.SearchUsers`, `user_service.SearchUsers`, `queries/user_search.sql`. (Obert com a DP-035; renumerada a DP-037 en fer merge de `develop`, que ja tenia DP-033 a DP-036.)
- **Dubte / supòsits:**
  1. Es cerca per «conté» (`ILIKE`) al nom d'usuari i al nom, sense distingir majúscules. Els comodins `%`, `_` i `\` es cerquen literalment. Ordre: nom d'usuari exacte, després els que comencen pel text i al final la resta, alfabèticament. No s'ignoren els accents (`Nuria` no troba `Núria`).
  2. «Comptes desactivats»: s'exclouen tots els usuaris amb un `AccountState` diferent d'`Active` (ara és l'únic estat que existeix; R6 en preveu el `Desactivat`).
  3. Bloquejos: s'exclouen en els dos sentits (qui ha bloquejat l'usuari que cerca i qui ell ha bloquejat), com fa `IsBlocked`.
  4. L'usuari que cerca no surt als resultats. Cal identitat (`401` sense).
  5. Paginació com `GET /events/search`: `pageNum` (des de 0) i `pageSize` (20 per defecte, màxim 50); `q` buit retorna una pàgina buida en lloc de tots els usuaris; `q` de més de 100 caràcters → `400`.
  6. Cada resultat només porta `id`, `username`, `name` i `profile-picture` (ni el correu ni si és amic).
  7. SQL no provat contra PostgreSQL/Supabase; `user_search.sql.go` s'ha escrit a mà en el format de sqlc.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-038 · Estat d'un esdeveniment no oficial i modificació amb `PUT /events/{id}`
- **Estat:** oberta
- **Obert per:** 2026-11-18 · Claude Code (Opus 5.5) · branca `feature/modificar-esdeveniment` (Taiga #188)
- **Context:** (Obert com a DP-034 amb la migració 000017; renumerades a DP-038 i 000018 en fer merge de `develop`, que ja les tenia ocupades.) La tasca demana verificar l'«estat actiu», però la BD no tenia cap estat per als NoOficials (`20-model-domini.md`: `NoOficial.estat: Retirat, … TODO(confirmar)`). L'equip (Arnau) ha triat afegir una columna d'estat.
- **Dubte / supòsits:**
  1. Migració 000018: enum `UnofficialEventState ('Active', 'Withdrawn')` i columna `UnofficialEvents.State NOT NULL DEFAULT 'Active'`. Els esdeveniments existents queden `Active`. Els oficials no en tenen (R7: mai no poden estar Retirats).
  2. `PUT /events/{id}` respon `409 Conflict` si l'estat no és `Active`. L'ordre de comprovacions és: autoria i tipus (`403`) → estat (`409`) → camps i R5 (`400`).
  3. De moment cap endpoint no canvia l'estat (la retirada arribarà amb les denúncies, R7). L'estat no surt a les respostes JSON dels esdeveniments.
  4. No s'ha bloquejat `POST /events/{id}/image` ni `DELETE /events/{id}` per als esdeveniments retirats, ni s'amaguen dels llistats: fora de l'abast de #188.
  5. Pendent de confirmar si cal un estat `Cancelled` (cancel·lat pel creador), diferent de `Withdrawn`. → La Tasca #180 l'afegeix (DP-041).
  6. La migració i la consulta `GetUnofficialEventState` no s'han provat contra PostgreSQL/Supabase; `events.sql.go` s'ha escrit a mà en el format de sqlc (sense regenerar, pels ajustos manuals del fitxer).
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** equip (backend + frontend)
- **Resolució:**

### DP-039 · Recuperar la contrasenya: endpoint públic `POST /auth/password-reset`
- **Estat:** oberta
- **Obert per:** 2026-11-18 · Claude Code (Opus 5.5) · branca `feature/recuperar-contrasenya` (Taiga #118, part backend)
- **Context:** (Obert com a DP-033; renumerada a DP-039 en fer merge de `develop`, que ja tenia fins a la DP-038.) `password_reset_handler`, `password_reset_service`, `firebase_client.IdentityToolkitClient`. La tasca és sobretot de frontend (`sendPasswordResetEmail`, `ForgotPasswordScreen`); el backend hi afegeix poder fer-ho amb el nom d'usuari i un límit al servidor.
- **Dubte / supòsits:**
  1. Endpoint **públic** (qui ha oblidat la contrasenya no té sessió), muntat fora del router com el de Mobilicat. Cos `{"identifier": "..."}`: si conté `@` és un correu; si no, un nom d'usuari que es busca a la BD per obtenir-ne el correu.
  2. **Anti-enumeració:** respon `204` tant si el compte existeix com si no. Amb un nom d'usuari, els límits i els errors de Firebase tampoc no es comuniquen (un nom inexistent no arriba mai a Firebase). Una fallada de xarxa amb Firebase sí que retorna `502`.
  3. Enviament amb l'API REST de Firebase Auth (`accounts:sendOobCode`, `PASSWORD_RESET`) i la **Web API key** (`FIREBASE_WEB_API_KEY`; sense ella → `503`). L'Admin SDK no serveix: només genera l'enllaç i caldria un servei de correu propi. El correu i la pàgina de l'enllaç són les plantilles de Firebase, en català (`X-Firebase-Locale: ca`).
  4. Límit: 1 petició per identificador i per correu cada 60 s (`429` + `Retry-After`), en memòria (es perd en reiniciar i no es comparteix entre instàncies). Sense límit per IP. Firebase hi afegeix les seves quotes.
  5. El frontend (`FirebaseAuthRepository.sendPasswordResetEmail`) hauria de cridar aquest endpoint en lloc de `FirebaseAuth.sendPasswordResetEmail`, per poder acceptar el nom d'usuari. Pendent de la part FE.
  6. No provat contra Firebase real (sense la clau a l'entorn de desenvolupament): només tests unitaris amb `httptest`.
- **Opcions:** A) aquest endpoint · B) només el SDK de Firebase al client (sense nom d'usuari ni límit propi).
- **Provisionalment:** A.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-040 · Xat d'esdeveniment: taula `EventMessages` i `GET`/`POST /events/{id}/messages`
- **Estat:** oberta
- **Obert per:** 2026-11-19 · Claude Code (Opus 5.5) · branca `feature/xat-esdeveniment` (Taiga #147)
- **Context:** `message_handler`, `message_service`, `queries/event_messages.sql`, migració 000019. La tasca demanava una migració de la taula `event_messages`, però la taula ja existia com a `Messages` (migració 000001) amb la PK `(UserID, Event, DateSent)`, que és la clau R1 (autor + esdeveniment + dataEnviament). L'equip (Arnau) ha triat reanomenar-la.
- **Dubte / supòsits:**
  1. Migració 000019: `Messages` → `EventMessages` (com `EventRatings`), `DateSent DEFAULT now()` i índex `(Event, DateSent DESC)`. Es mantenen la PK i les FK. S'han actualitzat `DeleteUserMessages` i `DeleteEventMessages`.
  1b. **Compressió del text** (demanada per l'equip per estalviar espai a la BD): Postgres només comprimeix files de més d'~2 kB, que un missatge no arriba a fer, així que la comprimeix l'API (`message_codec.go`). `Message` passa a `bytea`: 1 byte de format (`0` = text UTF-8 tal qual, `1` = DEFLATE de `compress/flate`, sense dependències noves) + el text. Només es comprimeix si ocupa menys; els missatges curts (menys d'~100 bytes, la majoria) queden igual (+1 byte). Els llargs ocupen aproximadament la meitat. Conseqüències: el text ja no es pot llegir ni cercar des de SQL o el panell de Supabase (afecta, p. ex., futures denúncies de missatges); la BD només limita la mida (2..4001 bytes) i el límit de 1–1000 caràcters el comprova l'API; les files existents es converteixen al format `0`; el `down` no pot descomprimir els missatges en format `1` i els substitueix per `[missatge comprimit]`. L'alternativa que estalviaria més espai és esborrar els xats dels esdeveniments acabats fa N dies (tasca diària a `runDaily`), pendent de decidir.
  2. R11 també s'aplica a la lectura: només els inscrits poden veure el xat (`403` en tots dos casos, com R10 a les valoracions). «Inscripció activa» = existeix una fila a `SignUp` (desapuntar-se l'esborra).
  3. El creador d'un esdeveniment no oficial no hi té accés si no s'hi ha inscrit.
  4. No es comprova l'estat de l'esdeveniment (`Withdrawn`), ni si ja ha acabat, ni els bloqueigs entre usuaris: tots els inscrits veuen tots els missatges.
  5. Paginació: del més nou al més antic, `pageNum` des de 0, `pageSize` 50 per defecte i màxim 100. Sense temps real (el client fa polling).
  6. Dos missatges del mateix usuari al mateix esdeveniment en el mateix microsegon violen la clau R1 → `409`.
  7. Cada missatge porta `user-id`, `username`, `name`, `profile-picture`, `event-id`, `text` i `date-sent`.
  8. La migració i les consultes no s'han provat contra PostgreSQL/Supabase; `event_messages.sql.go` s'ha escrit a mà en el format de sqlc.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-041 · Cancel·lar un esdeveniment no oficial (`POST /events/{id}/cancel`)
- **Estat:** oberta
- **Obert per:** 2026-11-19 · Claude Code (Opus 5.5) · branca `feature/cancellar-esdeveniment` (Taiga #180)
- **Context:** `EventHandler.CancelEvent`, `EventService.CancelUnofficialEvent`, `EventRepository.Cancel`, consultes `CancelUnofficialEvent` i `CreateCancellationNotifications`, migració 000020. (La DP-040 és a la PR #69 del xat.)
- **Dubte / supòsits:**
  1. Migració 000020: nou valor `Cancelled` a `UnofficialEventState` (diferent de `Withdrawn`, que és per denúncies, R7) i nou tipus de notificació `Cancelacio` a `NotificationTypes` (`NumNotificationTypes` passa a 4). El tipus s'insereix amb la migració perquè `NotificationRepository` esborra i recrea els tipus quan no n'hi ha `NumNotificationTypes`, i aquest esborrat fallaria si ja hi ha notificacions. El `down` no pot treure el valor de l'enum (Postgres no ho permet) i torna els cancel·lats a `Active`.
  2. Autoria com a `PUT`/`DELETE`: només el creador (`403` per a la resta i per als oficials). Ordre: autoria (`403`) → acabat (`409`) → estat (`409`).
  3. Només es pot cancel·lar un esdeveniment `Active` que no hagi acabat (darrer dia = `EndDate`, o `StartDate` si no en té, abans d'avui → `409`). Un esdeveniment en curs sí que es pot cancel·lar. Tornar-lo a cancel·lar dona `409`.
  4. El canvi d'estat i les notificacions van en una sola transacció; l'`UPDATE` només actua si l'estat encara és `Active`, així dues peticions alhora no generen notificacions duplicades.
  5. Es notifica un cop cada usuari inscrit, excepte el creador, amb el text «<títol> s'ha cancel·lat.» (en català, com els recordatoris). No es tenen en compte les preferències de notificació de `Settings`. Com que ja tenen una notificació de l'esdeveniment, `CreateReminders` ja no els enviarà el recordatori.
  6. Les inscripcions no s'esborren, l'esdeveniment segueix als llistats i el JSON de l'esdeveniment no mostra l'estat. Tampoc no es bloquegen les noves inscripcions, el xat ni la imatge d'un esdeveniment cancel·lat. Cal decidir-ho amb el frontend.
  7. Resposta `200 {event-id, state, notified-users}`. No hi ha manera de desfer la cancel·lació.
  8. La migració i les consultes no s'han provat contra PostgreSQL/Supabase; el codi de `sqlc` s'ha escrit a mà en el seu format.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-042 · Bloquejar i desbloquejar usuaris (`/users/{id}/block`, `GET /users/me/blocked`)
- **Estat:** oberta
- **Obert per:** 2026-11-19 · Claude Code (Opus 5.5) · branca `feature/usuaris-bloquejats` (Taiga #108, part backend)
- **Context:** `user_handler/blocked_users.go`, `user_service/blocked_users.go`, `postgres/blocked_users.go`, `queries/blocked_users.sql`, migració 000021. La taula `Blocked` ja existia (000001: PK `(BlockingUser, BlockedUser)`, `CHECK` R2, FK a `Users` des de la 000009) i ja s'usava a `IsBlocked` (sol·licituds d'amistat) i a la cerca d'usuaris, però no hi havia cap endpoint. (La DP-040 i la DP-041 són a les PR #69 i #70.)
- **Dubte / supòsits:**
  1. Migració 000021: columna `DateBlocked` (`DEFAULT now()`; els bloquejos existents queden amb la data de la migració) per ordenar la llista, i índex per `BlockedUser`.
  2. R3: en bloquejar, en una sola transacció s'elimina l'amistat (les dues files de `Friendships`) i les sol·licituds `Pending` o `Accepted` entre tots dos passen a `Rejected` (es manté la `DateResolved` de les acceptades). No s'esborren perquè les notificacions socials les referencien.
  3. En desbloquejar no es recupera l'amistat: cal tornar a enviar una sol·licitud.
  4. L'usuari bloquejat no rep cap notificació. Bloquejar no amaga els esdeveniments, el xat ni les notificacions antigues de l'altre usuari; només impedeix les sol·licituds d'amistat (`IsBlocked`) i el treu de la cerca.
  5. Rutes: `POST /users/{id}/block` (`201` amb l'usuari bloquejat), `DELETE /users/{id}/block` (`204`) i `GET /users/me/blocked` (`200`, del més recent al més antic, sense paginació). `{id}` accepta UUID, Firebase UID o nom d'usuari, com `unfriend`. Codis: `400` R2, `404` usuari inexistent o no bloquejat, `409` ja bloquejat.
  6. La migració i les consultes no s'han provat contra PostgreSQL/Supabase; `blocked_users.sql.go` s'ha escrit a mà en el format de sqlc.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-043 · Canviar idioma (#28): idioma de l'usuari i traducció d'esdeveniments
- **Estat:** oberta
- **Obert per:** 2026-11-19 · Claude Code (Sonnet 5.5) · branca `feature/change-language-28` (Taiga #28)
- **Context:** `translate_client`, `translation_service`, `translation_repository`, `handlers/language`, `queries/translations.sql`, migració 000022. Decisions ja preses per l'usuari: idiomes `ca`/`es`/`en` (original `ca`), servei traductor extern sobre Cloudflare Workers AI (`TRANSLATOR_URL`), cau a BD i idioma persistit a `Settings`.
- **Dubte / supòsits:**
  1. **Numeració de la migració:** `develop` ja té fusionades `000020` (`event_cancellation`) i `000021` (`blocked_users`), així que s'usa **000022** (`supabase/migrations/20261119000022_event_translations.sql`). Compte amb la PR #73 (`feature/settings-improvements`), que també porta una migració pendent: si es fusiona abans, caldrà renumerar aquesta.
  2. **Idioma de resolució:** `?lang` guanya; sense ell, l'idioma de `Settings` de l'usuari autenticat; sense usuari (o si no es pot identificar o llegir), l'original. Un usuari sense `POST /auth/sync` no rep `401` per això: simplement veu l'original. `ca` no crida mai el traductor.
  3. **Camps traduïts:** arran de la revisió de la PR #74 (Lluc), es tradueixen `title`, `subtitle`, `description`, `schedule`, `tickets` i `url-names` (separat per comes per traduir cada nom d'enllaç sense trencar la correspondència amb `related-urls`). Pendent de confirmar amb el frontend si s'acaba mostrant `subtitle`. `place`, `municipality`, `region`, `address`, etc. queden sense traduir. L'esdeveniment retornat per `POST`/`PUT /events` tampoc es tradueix (és el text de l'autor).
  4. **Cau:** `EventTranslations(Event, Lang, Title, Subtitle, Description, Schedule, Tickets, URLNames, SourceHash, UpdatedAt)` amb PK `(Event, Lang)`, `ON DELETE CASCADE` i RLS. `SourceHash` = SHA-256 dels sis camps originals separats per `0x00`; un canvi a qualsevol dels sis camps invalida la traducció. No hi ha caducitat ni esborrat de traduccions antigues.
  5. **Fallada de traducció:** es retorna el text original, es registra (`slog`) i no es desa res a la cau; amb `TRANSLATOR_URL` buit el client queda desactivat (`ErrDisabled`, no es registra per petició). Els esdeveniments sense cau d'una petició es divideixen en lots de 3 (fins a 4 en paral·lel, màxim 20 per petició) amb deduplicació dels textos repetits dins de cada lot i un timeout compartit de 12 s (`context.WithTimeout`, per sota del `WriteTimeout` de 15 s del servidor); cada lot es desa a la cau independentment, de manera que si un lot falla o expira, la resta igualment es tradueixen i es guarden. Cas del mapa: `GET /events/map` no té `LIMIT`, així que amb un mapa fred es tradueixen els primers 20 esdeveniments sense cau (els que ja són a la cau sempre es serveixen traduïts); la resta apareixen en català fins que la cau s'escalfi. El timeout de 15 s del client HTTP és només el límit superior.
  6. **Camp `Settings.Language`:** no s'ha regenerat `sqlc` (`models.go` té ajustos manuals). Les consultes noves són a `queries/translations.sql` i `sqlc/translations.sql.go` s'ha escrit a mà; `GetSettings`/`UpdateUserSettings` generats no coneixen la columna, així que `UserRepository` fa una consulta addicional per llegir-la, i en desar-la usa la consulta manual `UpdateSettings` (un sol `UPDATE` amb `ShowPlans`, `NotificationBefore` i `Language`, atòmic). Si es regenera `sqlc`, `Setting` guanyarà el camp `Language` i les consultes duplicades es podran fusionar.
  7. **Fora d'abast:** el xatbot (`assistant_handler`/MCP) no tradueix els esdeveniments ni respon en l'idioma de l'usuari; la qualitat de totes les traduccions automàtiques no s'ha revisat manualment.
  8. **Millores conegudes, no corregides:** (a) cada petició autenticada fa cerques d'usuari addicionals (resolució de Firebase UID → UUID i lectura de `Settings.Language`); (b) `language.Resolve` accepta el fallback de capçalera `X-User-ID` (com la resta de l'API en mode de proves), cosa que permetria llegir l'idioma d'un altre usuari amb només conèixer el seu ID; (c) no hi ha `singleflight`: peticions concurrents pels mateixos esdeveniments sense cau tradueixen i desen dues vegades.
  9. La versió inicial (títol i descripció) es va verificar contra PostgreSQL local i Cloudflare Workers AI; l'ampliació amb `Subtitle`, `Schedule`, `Tickets` i `URLNames` està verificada amb tests unitaris i queda pendent de re-provar contra PostgreSQL/Supabase i el Worker real.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors.
- **Qui decideix:** frontend + backend
- **Resolució:**

### DP-044 · Notificacions push amb FCM (#275): contracte, outbox i preferència global
- **Estat:** oberta
- **Obert per:** 2026-11-20 · Claude Code (Opus 5.5) amb confirmació de l'usuari · branca `feature/push-notifications` (Taiga #275)
- **Context:** `push_handler`, `push_service`, `push_repository`, `firebase_client/push.go`, `queries/push_notifications.sql`, migració 000024, `cmd/app/main.go`. Decisions ja preses per l'usuari: totes les notificacions de la bústia (`Notifications`) s'envien també com a push; interruptor global `Settings.PushEnabled`; outbox cada 30 s amb caducitat de 24 h; títol segons el tipus. La part Flutter és d'un altre equip.
- **Dubte / supòsits:**
  1. **Contracte (pendent de validar amb el frontend):** `PUT /users/me/devices` amb `{"token", "platform": "android"|"ios"|"web"}` → `204` (upsert: un token ja registrat passa a l'usuari actual, p. ex. un mòbil compartit); `DELETE /users/me/devices/{token}` → `204` sempre (idempotent, per al tancament de sessió). Push: `notification.title`/`body` i `data` (tot cadenes) amb `notification-id`, `type` (nom del `NotificationType`), `event-id` i `friend-id` quan n'hi ha. `PATCH /users/me` accepta `push-enabled` i `GET /users/me` el retorna només al perfil propi.
  2. **Outbox:** `Notifications.PushedAt` marca el que ja s'ha enviat; un ticker (`PUSH_DISPATCH_INTERVAL`, per defecte 30 s) reclama lots de 100 amb `FOR UPDATE SKIP LOCKED` dins d'una transacció, envia i marca. No s'ha tocat cap servei que crea notificacions. La migració marca l'històric com a enviat. Es marquen sense enviar: usuaris amb `PushEnabled=false`, usuaris sense dispositius i les pendents de més de 24 h (`MaxPushAge`). Una fallada general de FCM o només errors transitoris deixen la notificació pendent per al següent tic (lliurament *at-least-once*: si falla el `MarkPushed` després d'enviar, es pot repetir un push).
  3. **Tokens no vàlids:** `UNREGISTERED`, `INVALID_ARGUMENT` i `SENDER_ID_MISMATCH` esborren el token. Per evitar que un payload invàlid esborri tokens bons, el cos es retalla a 1000 caràcters (`MaxBodyLength`).
  4. **Token o FID:** el SDK v4.22.0 declara obsolets els *registration tokens* en favor dels *Firebase Installation IDs*. Flutter encara obté tokens (`getToken()`) i FCM els accepta, així que el backend accepta tots dos: els que contenen `:` s'envien com a token (amb `//nolint:staticcheck`) i la resta com a FID. Quan l'app registri FIDs, es pot treure el camí dels tokens.
  5. **Títols:** Social → «Social», Recordatori → «Recordatori», Recomanacio → «Recomanació», Cancelacio → «Esdeveniment cancel·lat», altres → «AProp!». El cos és el `Text` de la notificació, que es desa en català: no es tradueix segons `Settings.Language` (relacionat amb DP-043).
  6. **Relació amb la PR #73 (US #19 Configurar les notificacions):** aquella PR filtra en crear les notificacions segons `AllowedNotifications`; com que el push replica la bústia, el filtre s'hi aplica sol. `PushEnabled` és independent (la notificació arriba a la bústia però sense push). Totes dues toquen `UpdateProfile`, `user_repository.UpdateSettings` i `openapi.yaml`: caldrà resoldre conflictes, i el `GET` de configuració de la #73 hauria d'incloure `push-enabled`.
  7. **Numeració de la migració:** s'usa **000024** perquè la PR #76 (`feature/restringir-fotos-perfil`) ja porta la 000023. `golang-migrate` no aplica una versió inferior a l'actual: cal fusionar la #76 abans que aquesta (o renumerar). Les PR #73 i #75 porten una 000020 que també caldrà renumerar.
  8. **`sqlc`:** `sqlc generate` falla al repositori (la 000020 fa `ALTER TYPE UnofficialEventState`, que sqlc no coneix) i `models.go` té ajustos manuals; s'ha generat en una còpia temporal i només s'ha copiat `sqlc/push_notifications.sql.go`. `PushEnabled` es llegeix i es desa amb consultes pròpies (com `Language`), i `UpdateSettings` ara fa les seves sentències en una transacció.
  9. **ADR:** si l'equip valida l'outbox, cal un ADR-0006.
- **Opcions:** —
- **Provisionalment:** els supòsits anteriors, amb `TODO(DP-044)` a `push_service.go`, `firebase_client/push.go` i `user_repository.go`.
- **Qui decideix:** frontend (contracte i token/FID) + backend + equip (outbox, ADR)
- **Resolució:**

## Resoltes

(cap)

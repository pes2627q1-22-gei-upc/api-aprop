---
description: "Convencions en editar codi Go del backend (cmd/, internal/)."
globs: **/*.go
alwaysApply: false
---

# Codi Go (cmd/, internal/)

Complementa `AGENTS.md` (arquitectura i límits) amb les convencions en tocar codi Go:

- **Arquitectura en capes:**
  - `handlers/`: només transport HTTP, parsing de paràmetres, validació d'entrada i resposta JSON amb `handlers/response`. Cap accés directe a BD ni lògica de negoci pesada.
  - `services/`: lògica de negoci pura darrere d'interfícies (`interface.go`). No importar `net/http` ni barrejar estructures de transport.
  - `repositories/`: persistència darrere d'interfície. Consultes tipades a `internal/repositories/postgres/sqlc`.
  - `clients/`: consum de serveis externs darrere d'interfície (`gencat_client`, `groq_client`, etc.).
- **Errors i respostes:**
  - Utilitza errors embolicats (`fmt.Errorf("...: %w", err)`).
  - Retorna errors d'usuari a través d'`internal/handlers/response`.
- **JSON:**
  - Totes les claus JSON han d'estar en **kebab-case** (`start-date`, `place-id`, `tickets-url`).
- **Concurrència i context:**
  - Passa sempre `context.Context` com a primer paràmetre a mètodes de repositori i crides HTTP externes.
- **Transaccions:**
  - Operacions amb múltiples escriptures correlacionades han d'executar-se sota una transacció `pgx.Tx`.

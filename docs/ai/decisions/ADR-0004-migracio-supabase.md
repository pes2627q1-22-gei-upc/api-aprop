# ADR-0004 · Migrar la base de dades PostgreSQL a Supabase gestionat amb RLS

- **Estat:** acceptada
- **Data:** 2026-10-23 · **Autoria:** Wenqiang He · **Taiga:** #224

## Context

Inicialment es contemplava una instància autogestionada o RDS de PostgreSQL bàsica. Calia una solució al cloud gestionada, amb un procés àgil de migracions, que complís els requisits no funcionals de seguretat i privacitat de les dades (NFR) i facilités la gestió d'usuaris i permisos.

## Decisió

Utilitzar **Supabase** com a instància PostgreSQL gestionada al cloud, aprofitant el Supabase CLI per al versionat de migracions a `supabase/migrations/`, connectant l'API Go mitjançant el driver d'alt rendiment `pgxpool` i activant **Row Level Security (RLS)** a totes les taules públiques amb polítiques de lectura/escriptura restringides.

## Alternatives considerades

- **PostgreSQL local / contenidor aïllat:** Dificultat per sincronitzar dades i entorns entre tots els membres de l'equip i dificultat per fer proves conjuntes amb el frontend.
- **AWS RDS pur:** Cost i complexitat addicional de gestió en comparació amb la facilitat i funcionalitats integrades que ofereix Supabase (UI, backups, gestió de connexions i seguretat).

## Conseqüències

- (+) Infraestructura gestionada, alta disponibilitat i entorn accessible per a tots els membres de l'equip.
- (+) Seguretat reforçada a nivell de fila amb polítiques RLS (migració 000005).
- (+) Eina CLI estàndard per a desplegament i reproduïbilitat de migracions.
- (−) Cal mantenir les cadenes de connexió segures mitjançant variables d'entorn (`DB_HOST`, `DB_PORT`, `DB_PASSWORD`).

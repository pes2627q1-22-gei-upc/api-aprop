# ADR-0005 · Implementació de l'assistent cultural amb Groq AI i protocol MCP

- **Estat:** acceptada
- **Data:** 2026-10-24 · **Autoria:** Wenqiang He · **Taiga:** #225

## Context

L'assistent cultural (xatbot) ha d'oferir respostes en llenguatge natural en català, interpretar paràmetres de cerca (categoria, municipi, dates relatives) i recomanar exclusivament activitats del catàleg real sense al·lucinacions, amb una latència inferior a 10 segons (NFR).

## Decisió

Substituir la dependència de Gemini directe per **Groq AI** (model Llama 3.3 70B Versatile), utilitzant la capacitat nativa de *tool calling* (function calling) per consultar el catàleg mitjançant un servidor MCP independent (`cmd/mcp`). L'API principal (`cmd/app`) actua com a orquestrador entre l'usuari, Groq AI i el servidor MCP.

## Alternatives considerades

- **Google Gemini API directa:** Menor velocitat d'inferència en comparació amb l'arquitectura LPU de Groq i costos/límits de quota més restrictius durant el desenvolupament.
- **Consultes SQL directes generades per l'LLM:** Risc d'errors de sintaxi SQL, injeccions o accés a taules no autoritzades. El protocol MCP encapsula les consultes en eines segures i tipades (`search_events`).

## Conseqüències

- (+) Temps de resposta dràsticament reduïts (respostes en 1-2 segons).
- (+) Zero al·lucinacions d'esdeveniments: només es recomanen activitats reals retornades pel catàleg.
- (+) Desacoblament net gràcies al protocol estàndard MCP.
- (−) Requereix l'execució del binari addicional `cmd/mcp` (incorporat al mateix contenidor Docker).

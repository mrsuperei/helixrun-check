# HelixRun Agent Contract (Codex / Code Agents)

Dit document beschrijft hoe generieke code-agents (zoals Codex of vergelijkbare LLM-code agents) zich binnen de HelixRun-repository moeten gedragen.

Doelen:

- HelixRun en `trpc-agent-go` correct begrijpen en respecteren
- Idiomatische, veilige en onderhoudbare Go-code genereren
- Taken en documentatie consequent bijwerken in `/planning` en relevante `README.md`-bestanden
- Geen project-specifieke logica hard-coderen als dit via configuratie kan

Gebruik dit bestand als primair contract voor elke agent die code of configuratie in deze repo genereert of wijzigt.

---

## 1. Rol van de agent binnen HelixRun

1. De agent is een **Go-georiënteerde systeem- en code-assistent** die:
   - Agents, graphs, tools, runners en modellen implementeert of wijzigt op basis van `trpc-agent-go`
   - User prompts omzet naar concrete taken in `/planning` (zie sectie 6)
   - De relevante documentatie (`README.md`-bestanden) bijwerkt of aanmaakt

2. De agent MOET:
   - Go 1.21+ gebruiken en idiomatische Go-regels volgen
   - Bestaande projectarchitectuur respecteren (structuur, patterns, naming)
   - `context.Context` consistent gebruiken en propagateren
   - Concurrency veilig toepassen (geen goroutine leaks, duidelijke lifecycle)
   - Configuraties zo veel mogelijk **data-gedreven** houden (JSON/DB/YAML), niet hard-coded

3. De agent MAG:
   - Kleine refactors uitvoeren om code beter testbaar of uitbreidbaar te maken
   - Nieuwe `README.md`-bestanden aanmaken als die nodig zijn om gebruik / inrichting uit te leggen
   - Taken opsplitsen of herformuleren zolang de planning in `/planning` consistent blijft

4. De agent MAG NIET:
   - Niet-bestaande of verouderde `trpc-agent-go` APIs gebruiken
   - Grote, risicovolle rewrites doen zonder duidelijke reden en zonder dit in planning + docs te beschrijven
   - Willekeurig nieuwe top-level directories toevoegen buiten de bestaande architectuur

---

## 2. Architectuur: HelixRun en tRPC-Agent-Go

HelixRun gebruikt `trpc-agent-go` als kern van het agent- en workflow-systeem. Belangrijke concepten:

- **Model-laag**  
  `trpc.group/trpc-go/trpc-agent-go/model`  
  Abstractie over LLM-backends (OpenAI-compatibel, lokale modellen, andere providers).

- **Agent-laag**  
  `trpc.group/trpc-go/trpc-agent-go/agent`  
  Beschrijft hoe een agent input aanneemt, redeneert en events uitstuur naar de runner.

- **Runner**  
  `trpc.group/trpc-go/trpc-agent-go/runner`  
  Verantwoordelijk voor:
  - Sessies aanmaken/laden
  - Agents of graphs uitvoeren
  - `event.Event`-stromen naar de UI/clients
  - Het opslaan van history in de session backend

- **Graph**  
  `trpc.group/trpc-go/trpc-agent-go/graph`  
  State graph workflows (nodes, edges, HITL, branching, cycles).

- **Multi-Agent**  
  Orkestratie van meerdere agents (bijv. planner + solver, parallelle agents).

- **Tools**  
  `trpc.group/trpc-go/trpc-agent-go/tool` en `tool/agent`  
  - Function tools (Go-functies)
  - MCP tools
  - Agents als tools in andere agents of graphs

- **Sessions / Memory**  
  `trpc.group/trpc-go/trpc-agent-go/session`  
  Beheer van chatgeschiedenis, samenvattingen en sessie-IDs.

- **Knowledge / RAG**  
  `trpc.group/trpc-go/trpc-agent-go/knowledge`  
  Knowledge en RAG-integratie (vector search, doc-injectie).

- **Events & Telemetry**  
  `trpc.group/trpc-go/trpc-agent-go/event`  
  Gestandaardiseerde event-stream voor LLM-output, tools, errors, graph-transities, etc.

De agent moet deze bestaande lagen benutten en niet proberen om eigen frameworks voor dezelfde problemen te bouwen.

---

## 3. Projectstructuur die de agent moet volgen

Tenzij de repo expliciet anders is ingericht, hanteert HelixRun de volgende conventie:

- `/cmd/server`  
  Entrypoint voor de server (HTTP/gRPC), wiring van Runner(s), agents, graphs en configuratie.

- `/internal/agents`  
  Agent-definities, factories en agent-registratie.

- `/internal/graph`  
  Graph-schema's, node-implementaties, graph-bouwers en graf-registratie.

- `/internal/model`  
  Modelinstellingen en providers, bijv. OpenAI/Ollama/andere.

- `/internal/http`  
  HTTP-handlers, router, SSE/WS endpoints, auth-middleware.

- `/internal/agui`  
  Integratie met AG-UI en/of HelixRun UI, inclusief event-streaming.

- `/internal/store`  
  Data-opslag (PostgreSQL/Redis), sessie-opslag, artifacts, knowledge-stores.

- `/internal/telemetry`  
  Tracing, metrics, logging helpers en integratie met observability tools.

- `/pkg/utils`  
  Algemeen herbruikbare helpers, geen domeinspecifieke logica.

- `/planning`  
  Planning- en ontwerpdocumenten, waaronder planning van taken en roadmap.

De agent moet nieuwe code onder een passend `internal`-pakket plaatsen en consistent zijn met de bestaande layout.

---

## 4. Standaard Go-richtlijnen

De agent moet standaard Go-richtlijnen respecteren, zowel qua stijl als architectuur.

### 4.1 Taalversie en modules

- Gebruik **Go 1.21+**
- `go.mod` moet geldig zijn en module path consistent met de repo-root.
- Nieuwe externe dependencies beperken en goed motiveren in comments/README.

### 4.2 Naamgeving en structuur

- Pakketnamen kort en beschrijvend (`graph`, `agents`, `store`, `telemetry`, `http`).
- Functies en types met duidelijke namen; exported types en functies beginnen met een hoofdletter.
- Vermijd “god types” of te grote bestanden; splits logische componenten op.

### 4.3 Errors en logging

- Retourneer errors expliciet: `(..., error)` of alleen `error`.
- Wrap errors waar context belangrijk is:

  ```go
  if err != nil {
      return nil, fmt.Errorf("failed to load session %s: %w", sessionID, err)
  }
  ```

- Logging niet in library-functies hard-coderen; gebruik bij voorkeur een geinjecteerde logger of telemetrystelsel.
- Geen panics gebruiken voor normale foutafhandeling.

### 4.4 Context

- Elke functie die I/O doet, een LLM aanroept, met DB/Redis werkt of langdurig draait, moet `ctx context.Context` accepteren.
- Context wordt doorgegeven naar onderliggende calls en mag niet genegeerd worden.
- Bij blocking loops of worker-goroutines moet `ctx.Done()` worden gecontroleerd:

  ```go
  select {
  case <-ctx.Done():
      return ctx.Err()
  case msg := <-ch:
      // verwerk msg
  }
  ```

### 4.5 Concurrency en goroutines

- Goroutines alleen starten als dat nodig is, en altijd met een duidelijke exit-conditie (bv. context cancel).
- Geen onbeperkte goroutine-creatie in request-handlers.
- Gebruik evt. worker pools of bounded channels als er veel parallel werk is.

---

## 5. Integratie met tRPC-Agent-Go

### 5.1 Imports (kernpakketten)

De agent hoort minimaal bekend te zijn met de volgende imports:

```go
import (
    "trpc.group/trpc-go/trpc-agent-go/agent"
    "trpc.group/trpc-go/trpc-agent-go/model"
    "trpc.group/trpc-go/trpc-agent-go/runner"
    "trpc.group/trpc-go/trpc-agent-go/graph"
    "trpc.group/trpc-go/trpc-agent-go/event"
    "trpc.group/trpc-go/trpc-agent-go/tool"
    agenttool "trpc.group/trpc-go/trpc-agent-go/tool/agent"
    "trpc.group/trpc-go/trpc-agent-go/session"
    "trpc.group/trpc-go/trpc-agent-go/knowledge"
)
```

Voor concrete code moet de agent de bestaande imports in de repo controleren en daaraan alignen.

### 5.2 Runner

- De runner is het primaire entrypunt om agenten of graphs uit te voeren.
- Gebruik een gedeelde `runner.Runner` instance per configuratie; niet per request een nieuwe runner creëren zonder noodzaak.
- Runner beheert sessies en event-streaming richting UI.

### 5.3 Agents

- Een agent is doorgaans een configuratie rond een LLM-model met:
  - System prompt
  - Rolomschrijving
  - Tools
  - Event- en telemetry-hooks

- Agents kunnen worden ingepast in multi-agent opzetten (planner/worker, reviewer, etc.).

### 5.4 Graphs

- Gebruik `graph.NewStateGraph(schema)` om een type-veilige stategraph te definiëren.
- Typisch schema voor conversaties: `graph.MessagesStateSchema()`.
- Nodes en edges moeten duidelijk benoemd en gedocumenteerd zijn, zodat latere uitbreidingen eenvoudig zijn.
- Graphs horen in `/internal/graph` met per graph-type een apart bestand of pakket.

### 5.5 Tools en agent-tools

- Tools worden gebouwd met het `tool`-pakket; inputs/outputs moeten JSON-serialiseerbaar zijn.
- Als een agent vanuit een andere agent of graph moet worden aangeroepen, gebruik `tool/agent` zodat de agent als tool geregistreerd kan worden.
- Tool-gebruik moet zich ook manifesteren in events (`tool.request`, `tool.response`).

### 5.6 Knowledge / RAG

- Context (code, docs, eerdere gesprekken) wordt bij voorkeur via `knowledge`/RAG opgehaald in plaats van brute-force in prompts te dumpen.
- De agent moet, waar passend, `knowledge` integreren als tool in agents/graphs.

---

## 6. Planning & tasks in `/planning`

Planning en taakbeheer horen **niet** in dit `agents.md`-bestand, maar in een aparte directory:

- Directory: `/planning`
- Aanbevolen bestanden:
  - `/planning/todo.md`
  - `/planning/roadmap.md` (optioneel)
  - `/planning/design-*.md` (optioneel, voor deelontwerpen)

### 6.1 `/planning/todo.md` structuur

De agent MOET alle nieuwe taken en wijzigingen aan taken registreren in `/planning/todo.md`.

Aanbevolen vorm:

```markdown
# HelixRun TODO

## Backlog

- [ ] [ID:HR-001] [area:agents] Korte imperatieve beschrijving
  - Optionele detailregel 1
  - Optionele detailregel 2

## In Progress

- [ ] [ID:HR-010] [area:graph] Taak die momenteel actief is

## Done

- [x] [ID:HR-000] [area:meta] Voorbeeld van een afgeronde taak
```

Regels voor de agent:

1. Elke taakregel gebruikt checklist-syntax `- [ ]` of `- [x]`.
2. Elke taak heeft een ID `[ID:HR-xxx]` en een gebiedstag `[area:...]`.
3. De agent voegt nieuwe taken toe in “Backlog” (tenzij expliciet anders gevraagd).
4. Na afronden van een taak verplaatst de agent deze naar “Done” en zet `- [x]`.

### 6.2 Andere planning-bestanden

- `/planning/roadmap.md` kan gebruikt worden voor langere-termijn doelen.
- `/planning/design-*.md` kan gebruikt worden voor uitgebreide ontwerpen of ADR-achtige notities (Architectural Decision Records).

De agent mag nieuwe files in `/planning` aanmaken als dat helpt om de intentie van de wijzigingen duidelijk te documenteren.

---

## 7. README- en documentatiebeleid

De agent is verantwoordelijk voor het **consequent bijwerken** van documentatie bij elke relevante wijziging.

### 7.1 Hoofd-README

- `/README.md` beschrijft op hoog niveau:
  - Wat HelixRun doet
  - Hoe het gestart kan worden
  - De globale architectuur (runners, agents, graphs, UI)
- Bij grote architecturale wijzigingen moet de agent (indien nodig) `/README.md` updaten.

### 7.2 Module-specifieke READMEs

De agent hoort per relevant pakket of domein README’s te onderhouden, zoals:

- `/internal/agents/README.md`
- `/internal/graph/README.md`
- `/internal/http/README.md`
- `/internal/telemetry/README.md`
- `/internal/store/README.md`
- `/internal/model/README.md`
- `/planning/README.md` (uitleg over planningstructuur)

Richtlijnen:

1. Als een nieuwe agent, graph of tool wordt toegevoegd:
   - Voeg een korte beschrijving en gebruiksvoorbeeld toe in de relevante `README.md`.
2. Als een bestaand gedrag ingrijpend verandert:
   - Update de bestaande beschrijving en noem breaking changes of migratie-instructies.
3. Als er nog geen relevante `README.md` bestaat:
   - De agent MAG een nieuwe `README.md` aanmaken met minimaal:
     - Korte beschrijving van het package
     - Hoe het gebruikt wordt
     - Hoe het zich verhoudt tot `trpc-agent-go` (agents, graphs, runner, etc.).

### 7.3 Inline comments en doc comments

- `//` commentaar alleen gebruiken als het extra context toevoegt.
- Voor exported functies/types: `// Name ...` doc comments in Go-stijl.
- Geen commentaar dat alleen herhaalt wat de code al duidelijk maakt.

---

## 8. Events & Telemetry

Elke wijziging aan agents/graphs/tools moet rekening houden met event- en telemetryregels:

1. De event-stream moet minimaal de volgende typen blijven ondersteunen (indien van toepassing):
   - `run.start`, `run.end`, `run.error`
   - `chat.completion.chunk`, `chat.completion`
   - `tool.request`, `tool.response`, `tool.error`
   - Graph-specifieke events zoals node-enter/node-exit als de libs dat voorzien.

2. De agent mag nieuwe eventtypes toevoegen, maar mag bestaande, door de UI verwachte, types niet verwijderen zonder de UI en documentatie aan te passen.

3. Telemetry (traces, metrics):
   - Context moet span/trace ID’s kunnen dragen.
   - Modelcalls en tools zouden ideally duur (latency), token usage en foutstatistieken meten.

Als de agent nieuwe telemetry of eventtypes toevoegt, hoort dat kort beschreven te worden in een relevante `README.md` (bijvoorbeeld `/internal/telemetry/README.md` of `/internal/agui/README.md`).

---

## 9. Referentie: tRPC-Agent-Go documentatie

De agent moet de officiële documentatie van `trpc-agent-go` als primaire referentie gebruiken:

- Overzicht & handleiding  
  - https://trpc-group.github.io/trpc-agent-go/
  - https://deepwiki.com/trpc-group/trpc-agent-go

- API / Packages  
  - https://pkg.go.dev/trpc.group/trpc-go/trpc-agent-go

Belangrijke subsections:

- Agent:      https://trpc-group.github.io/trpc-agent-go/agent/
- Runner:     https://trpc-group.github.io/trpc-agent-go/runner/
- Graph:      https://trpc-group.github.io/trpc-agent-go/graph/
- MultiAgent: https://trpc-group.github.io/trpc-agent-go/multiagent/
- Model:      https://trpc-group.github.io/trpc-agent-go/model/
- Tool:       https://trpc-group.github.io/trpc-agent-go/tool/
- Session:    https://trpc-group.github.io/trpc-agent-go/session/
- Knowledge:  https://trpc-group.github.io/trpc-agent-go/knowledge/
- Event:      https://trpc-group.github.io/trpc-agent-go/event/

De agent mag patronen uit de officiële voorbeelden hergebruiken, maar moet altijd controleren of deze passen bij de bestaande HelixRun-structuur en -conventies.

# Parsing Approaches Comparison

## Single-Shot vs Multi-Stage vs Validated

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SINGLE-SHOT (Original)                                │
│                                                                              │
│   PRD ──────────────────────────────────────────────────────────► Issues    │
│         One massive LLM call generates everything                            │
│                                                                              │
│   Problems:                                                                  │
│   • Output token limits cause truncation                                     │
│   • Empty tasks[] arrays on large PRDs                                       │
│   • No parallelization                                                       │
│   • Retry = redo everything                                                  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                      MULTI-STAGE (Current)                                   │
│                                                                              │
│   PRD ──► Epics ──┬──► Tasks ──┬──► Subtasks ──► Issues                     │
│                   │            │                                             │
│                   │  parallel  │  parallel                                   │
│                   └────────────┴────────────                                 │
│                                                                              │
│   Benefits:                                                                  │
│   ✓ Smaller, focused prompts                                                 │
│   ✓ Parallel execution                                                       │
│   ✓ Retry individual stages                                                  │
│   ✓ Different models per stage                                               │
│                                                                              │
│   Still missing:                                                             │
│   • Validation between stages                                                │
│   • Tech stack awareness                                                     │
│   • Build verification                                                       │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                      VALIDATED (Proposed)                                    │
│                                                                              │
│   PRD ──► Tech ──► Epics ──► Review ──► Tasks ──► Review ──► Subtasks      │
│           Stack              │                    │                          │
│                              │                    │                          │
│                          ┌───┴───┐            ┌───┴───┐                      │
│                          │Inject │            │Inject │                      │
│                          │Setup  │            │Install│                      │
│                          │Epic   │            │Tasks  │                      │
│                          └───────┘            └───────┘                      │
│                                                                              │
│           ──► Build ──► Dependency ──► Issues                               │
│               Verify     Resolution                                          │
│               │                                                              │
│           ┌───┴───┐                                                          │
│           │Add    │                                                          │
│           │Missing│                                                          │
│           │Steps  │                                                          │
│           └───────┘                                                          │
│                                                                              │
│   Benefits:                                                                  │
│   ✓ All multi-stage benefits                                                 │
│   ✓ Tech-stack-aware task injection                                          │
│   ✓ Guaranteed buildable output                                              │
│   ✓ No missing setup/install steps                                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

## What Gets Injected Per Tech Stack

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ DETECTED: Next.js + Convex + Clerk + Telnyx                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│ AUTO-INJECTED EPIC 0: Project Initialization                                 │
│ ├── Task 0.1: Environment Setup                                              │
│ │   ├── Subtask: Create .env.local with required variables                   │
│ │   ├── Subtask: Document all required API keys                              │
│ │   └── Subtask: Create .env.example template                                │
│ │                                                                            │
│ ├── Task 0.2: Package Installation                                           │
│ │   ├── Subtask: Run bun install                                             │
│ │   └── Subtask: Verify all dependencies resolve                             │
│ │                                                                            │
│ ├── Task 0.3: Convex Setup                                                   │
│ │   ├── Subtask: Run npx convex dev (first time)                             │
│ │   ├── Subtask: Configure CONVEX_DEPLOYMENT                                 │
│ │   └── Subtask: Verify Convex dashboard access                              │
│ │                                                                            │
│ ├── Task 0.4: Clerk Setup                                                    │
│ │   ├── Subtask: Create Clerk application                                    │
│ │   ├── Subtask: Configure OAuth providers (if needed)                       │
│ │   ├── Subtask: Set CLERK_SECRET_KEY, NEXT_PUBLIC_CLERK_*                   │
│ │   └── Subtask: Test auth flow locally                                      │
│ │                                                                            │
│ └── Task 0.5: Telnyx Setup                                                   │
│     ├── Subtask: Create Telnyx Mission Control account                       │
│     ├── Subtask: Create TeXML application                                    │
│     ├── Subtask: Purchase or port phone number                               │
│     ├── Subtask: Configure webhook URLs (use ngrok for local)                │
│     └── Subtask: Test inbound/outbound call                                  │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ INJECTED AFTER SCHEMA TASKS:                                                 │
│ • "Regenerate Convex types" after any schema.ts change                       │
│ • "Run bun install" after any package.json change                            │
│ • "Restart dev server" after env variable changes                            │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ INJECTED BEFORE FEATURE TASKS:                                               │
│ • Verify previous setup tasks completed                                      │
│ • Check dev server is running                                                │
│ • Verify database connection                                                 │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Tech Stack Detection Rules

```yaml
# Proposed .prd-parser.yaml extension

tech_stack_rules:
  convex:
    detect:
      - file: "convex/schema.ts"
      - package: "convex"
    inject:
      - epic: "Project Setup"
        tasks:
          - "Initialize Convex (npx convex dev)"
          - "Configure environment variables"
      - after_pattern: "schema.ts"
        task: "Regenerate Convex types"

  clerk:
    detect:
      - package: "@clerk/nextjs"
      - prd_mentions: ["Clerk", "authentication"]
    inject:
      - epic: "Project Setup"
        tasks:
          - "Create Clerk application"
          - "Configure Clerk environment"
          - "Set up ClerkProvider"

  telnyx:
    detect:
      - package: "telnyx"
      - prd_mentions: ["Telnyx", "voice", "telephony"]
    inject:
      - epic: "Project Setup"
        tasks:
          - "Create Telnyx account"
          - "Configure phone number"
          - "Set up webhooks"
```

## Build Validation Questions

The final stage asks these questions:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       BUILD VALIDATION CHECKLIST                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│ Can an agent clone this repo and run it?                                     │
│                                                                              │
│ □ Is there a task to run `bun install` / `npm install`?                      │
│ □ Is there a task to set up environment variables?                           │
│ □ Is there a task to initialize the database?                                │
│ □ Is there a task to start the dev server?                                   │
│ □ Are all external service credentials documented?                           │
│                                                                              │
│ Will the code compile after all tasks are done?                              │
│                                                                              │
│ □ Are types generated before code that uses them?                            │
│ □ Are dependencies installed before imports?                                 │
│ □ Are migrations run before queries?                                         │
│ □ Is there a final "verify build" task?                                      │
│                                                                              │
│ Can the tests run?                                                           │
│                                                                              │
│ □ Is there a task to set up test environment?                                │
│ □ Are test dependencies installed?                                           │
│ □ Is there a task to run the test suite?                                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

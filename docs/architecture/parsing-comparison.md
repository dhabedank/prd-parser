# Parsing Approaches Comparison

## Single-Shot vs Multi-Stage

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SINGLE-SHOT PARSING                                   │
│                                                                              │
│   PRD ──────────────────────────────────────────────────────────► Issues    │
│         One LLM call generates everything                                    │
│                                                                              │
│   Best for:                                                                  │
│   ✓ Small PRDs (< 300 lines)                                                 │
│   ✓ Quick iterations                                                         │
│   ✓ Simple project structures                                                │
│                                                                              │
│   Limitations:                                                               │
│   • Output token limits can cause truncation                                 │
│   • May return empty tasks[] on large PRDs                                   │
│   • No parallelization                                                       │
│   • Retry = redo everything                                                  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                      MULTI-STAGE PARALLEL PARSING                            │
│                                                                              │
│   PRD ──► Epics ──┬──► Tasks ──┬──► Subtasks ──► Issues                     │
│                   │            │                                             │
│                   │  parallel  │  parallel                                   │
│                   └────────────┴────────────                                 │
│                                                                              │
│   Best for:                                                                  │
│   ✓ Large PRDs (≥ 300 lines)                                                 │
│   ✓ Complex project structures                                               │
│   ✓ Maximum reliability                                                      │
│                                                                              │
│   Benefits:                                                                  │
│   ✓ Smaller, focused prompts per stage                                       │
│   ✓ Parallel execution (3 epics, 5 tasks concurrent)                         │
│   ✓ Retry individual stages without redoing all                              │
│   ✓ Different models per stage (--subtask-model)                             │
│   ✓ Automatic retry on parse errors                                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Smart Detection (Default)

prd-parser automatically chooses based on PRD line count:

| PRD Size | Strategy | Reason |
|----------|----------|--------|
| < 300 lines | Single-shot | Faster, simpler |
| ≥ 300 lines | Multi-stage | More reliable |

Override with `--single-shot` or `--multi-stage` flags.

Adjust threshold with `--smart-threshold <lines>` (0 to disable).

## When to Use Each

### Use Single-Shot When:
- PRD is small and focused
- You want fastest possible execution
- Project has simple structure (few epics)

### Use Multi-Stage When:
- PRD is large or detailed
- You've seen empty tasks[] with single-shot
- You want maximum reliability
- You need different models for subtasks (cost optimization)

## Optional Validation

Both modes support `--validate` to run a final LLM review:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      VALIDATION PASS (--validate)                            │
│                                                                              │
│   After generation, asks LLM to review the complete plan:                    │
│   • Missing setup/initialization tasks?                                      │
│   • Backend built without UI to test it?                                     │
│   • Dependencies mentioned but not installed?                                │
│   • Acceptance criteria that can't be verified?                              │
│                                                                              │
│   Output: Warnings printed to console                                        │
│   Note: Advisory only - does not modify the generated plan                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

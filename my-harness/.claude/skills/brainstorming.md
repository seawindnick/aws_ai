# Brainstorming Skill

Generate a structured, divergent set of ideas on a topic. Use when the user wants to explore possibilities, break creative blocks, or map out options before committing to a direction.

## Behavior

When invoked with `/brainstorming <topic>`:

1. **Restate the challenge** in one sentence to confirm scope.
2. **Generate ideas** across at least 3 distinct angles or lenses (e.g. technical, UX, process, unconventional). Aim for 8–12 ideas total.
3. **Label each idea** with a short title and a one-sentence description.
4. **Flag standouts** — mark 2–3 ideas worth exploring further with a ★.
5. **Suggest next steps** — one sentence on how to narrow down or prototype.

## Format

```
## Brainstorm: <topic>

### Angle 1 — <label>
- **Idea title**: description
- ...

### Angle 2 — <label>
- ...

★ **Top picks**: <idea 1>, <idea 2>, <idea 3>

**Next step**: ...
```

## Notes

- Prioritize breadth over depth — this is divergent thinking, not design docs.
- Include at least one unconventional or "wild card" idea per session.
- Do not evaluate or critique ideas during generation; save that for follow-up.

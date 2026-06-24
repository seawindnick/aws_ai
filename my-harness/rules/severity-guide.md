# Severity Guide

Rules are classified by severity level. **MUST** rules are non-negotiable invariants — violations must be caught and fixed before merge, regardless of context.

## Severity Levels

| Level | Meaning |
|-------|---------|
| **MUST** | Hard requirement. No exceptions. Treat violations as blocking bugs. |
| **SHOULD** | Strong recommendation. Deviate only with explicit justification. |
| **MAY** | Optional guidance. Use when it fits the context. |

---

## Rules

### R1 — Error record data isolation `[MUST]`

**Rule:** 错题数据必须按 `userId` 隔离——学生只能读取、修改、删除属于自己的错题记录。

**Enforcement:**
- Every query that reads or writes error records MUST include a `userId` equality filter sourced from the authenticated session, never from client-supplied input.
- Any endpoint that returns error records MUST verify `record.userId == session.userId` before returning the record, even when the primary key was used to look up the item.
- Cross-user access MUST return `403 Forbidden`, never `404`, so that existence of another user's records is not leaked.

**Why this is MUST:** Exposing one student's wrong-answer history to another is a direct privacy violation and undermines test integrity.

---

### R2 — Bedrock response validation before use `[MUST]`

**Rule:** Bedrock 返回的响应必须先通过格式校验，再用于任何业务逻辑；`confidence` 字段缺失时必须按 `0` 处理。

**Enforcement:**
- Every Bedrock response MUST be validated against an expected schema (required fields, types) before its values are accessed.
- If `confidence` is absent or `null`, the consuming code MUST substitute `0` — it MUST NOT raise an exception, fall through with `undefined`, or silently skip the record.
- Validation failures (fields other than `confidence` that are missing or malformed) MUST be logged with the raw response and surfaced as an error to the caller rather than silently dropped.

**Why this is MUST:** An unvalidated Bedrock response can carry unexpected structure after model updates. Silent failures corrupt downstream scoring; a missing `confidence` treated as non-zero inflates student metrics.

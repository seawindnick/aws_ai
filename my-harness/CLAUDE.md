# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Rules

Project-wide engineering rules and their severity levels are maintained in [`rules/severity-guide.md`](rules/severity-guide.md). Read this file before writing any feature that touches error records or Bedrock responses.

Key MUST-level rules in that file:
- **R1** — Error record data MUST be isolated per `userId`; students may only access their own records.
- **R2** — Bedrock responses MUST be schema-validated before use; missing `confidence` MUST default to `0`.

## Backend Layer Conventions

Routes → Services → Models is a strict layering rule:
- Route handlers must stay thin: validate input, call service, return response.
- Never use boto3/DynamoDB directly in routes or services — always go through the model layer.
- Error responses: `{"error": "human-readable message"}` with appropriate HTTP status.
- List responses: `{"items": [...], "count": N}`.
- DynamoDB stores prices as `Decimal`; models serialize to `float` on read.

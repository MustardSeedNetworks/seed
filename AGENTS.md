# LLM Instructions (Codex CLI + Claude CLI)

These instructions apply to any coding or review task in this repo.

## Required Tool Use

Before writing or modifying code, you MUST consult the `language-specs` MCP server.

- Use `get_spec` for language/stdlib/formatter questions.
- Use `get_linter_rule` for lint guidance.
- Use `search_specs` if unsure.
- Use `list_available` if you need to discover topics.

If the task is not language-specific or no relevant spec exists, say so briefly and continue.

## Example

Prompt:
“Add error handling to Go HTTP client code.”

Expected first step:
Call `get_spec` with `language=go`, `category=stdlib`, `topic=net-http` before proposing code.

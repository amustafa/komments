Read the file `.komments/comments.json` in the project root. This file contains **komments** — non-inline code annotations left by the developer as structured metadata.

## What to do

1. Read `.komments/comments.json` and parse the JSON
2. Filter to **active** comments only (where `archived` is `false`)
3. For each active comment, interpret it as a contextual annotation at the specified file and location:
   - `position.type = "cursor"` → the comment applies at that specific line/column
   - `position.type = "range"` → the comment applies to the line range `start_line` through `end_line`
4. Read the referenced source code at each location to understand the full context
5. Address each comment — explain findings, suggest fixes, refactor code, or answer questions as appropriate
6. After fully addressing a comment, archive it using the `archive_comment` MCP tool with the comment's `id`

## Comment format

```json
{
  "id": 1,
  "timestamp": "2026-05-05T14:32:00Z",
  "file": "src/auth.go",
  "position": { "type": "cursor", "line": 42, "col": 10 },
  "text": "This retry logic silently swallows the error.",
  "archived": false
}
```

Treat each comment's `text` as a developer note, question, or instruction about the code at that location.

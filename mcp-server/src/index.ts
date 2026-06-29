import { execFileSync } from "child_process";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

const KOMMENTS_BIN = process.env.KOMMENTS_BIN || "komments";

function komments(...args: string[]): string {
  return execFileSync(KOMMENTS_BIN, args, {
    encoding: "utf-8",
    stdio: ["pipe", "pipe", "pipe"],
  }).trim();
}

const server = new McpServer({
  name: "komments",
  version: "1.0.0",
});

server.tool(
  "list_comments",
  "List code annotations (komments). Returns active comments by default. Pass include_archived to see all.",
  {
    include_archived: z
      .boolean()
      .optional()
      .describe("Include archived comments. Defaults to false."),
  },
  async ({ include_archived }) => {
    const args = ["list", "--json"];
    if (include_archived) args.push("--all");
    const output = komments(...args);
    return { content: [{ type: "text" as const, text: output }] };
  }
);

server.tool(
  "get_comment",
  "Get a single code annotation (komment) by its ID.",
  { id: z.number().describe("The comment ID to retrieve.") },
  async ({ id }) => {
    try {
      const output = komments("get", String(id), "--json");
      return { content: [{ type: "text" as const, text: output }] };
    } catch {
      return {
        content: [{ type: "text" as const, text: `Comment #${id} not found.` }],
        isError: true,
      };
    }
  }
);

server.tool(
  "archive_comment",
  "Archive a code annotation (komment) by its ID. Use this after addressing a comment.",
  { id: z.number().describe("The comment ID to archive.") },
  async ({ id }) => {
    try {
      const output = komments("archive", String(id));
      return { content: [{ type: "text" as const, text: output }] };
    } catch {
      return {
        content: [{ type: "text" as const, text: `Comment #${id} not found.` }],
        isError: true,
      };
    }
  }
);

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});

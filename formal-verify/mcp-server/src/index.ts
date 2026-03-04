import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import { dafnyVerify } from "./tools/verify.js";
import { dafnyCompile } from "./tools/compile.js";
import { dafnyCleanup } from "./tools/cleanup.js";

const server = new McpServer({
  name: "formal-verify-dafny",
  version: "1.0.0",
});

server.tool(
  "dafny_verify",
  "Verify Dafny source code. Writes source to a temp file, runs `dafny verify`, and returns structured results with errors/warnings.",
  {
    source: z.string().describe("Dafny source code to verify"),
  },
  async ({ source }) => {
    const result = await dafnyVerify({ source });
    return {
      content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
    };
  }
);

server.tool(
  "dafny_compile",
  "Compile verified Dafny source to Python or Go. Runs `dafny build`, strips Dafny runtime boilerplate, and returns clean output files.",
  {
    source: z.string().describe("Dafny source code to compile"),
    target: z
      .enum(["py", "go"])
      .describe("Target language: 'py' for Python, 'go' for Go"),
  },
  async ({ source, target }) => {
    const result = await dafnyCompile({ source, target });
    return {
      content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
    };
  }
);

server.tool(
  "dafny_cleanup",
  "Remove stale Dafny temp directories (older than 30 minutes) from /tmp.",
  {},
  async () => {
    const result = await dafnyCleanup();
    return {
      content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
    };
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

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import { dafnyVerify } from "./tools/verify.js";
import { dafnyCompile } from "./tools/compile.js";
import { dafnyCleanup } from "./tools/cleanup.js";
import { leanCheck } from "./tools/leanCheck.js";
import { leanRun } from "./tools/leanRun.js";
import { leanTest } from "./tools/leanTest.js";

export function createServer(): McpServer {
  const server = new McpServer({
    name: "crosscheck-dafny",
    version: "1.0.0",
  });

  server.tool(
    "dafny_verify",
    "Verify Dafny source code. Writes source to a temp file, runs `dafny verify`, and returns structured results with errors/warnings and difficulty metrics (solver time, resource count, proof hint count, trivial proof detection).",
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
    "Remove stale Dafny/Lean temp directories (older than 30 minutes) from /tmp.",
    {},
    async () => {
      const result = await dafnyCleanup();
      return {
        content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
      };
    }
  );

  server.tool(
    "lean_check",
    "Parse + typecheck Lean 4 source via `lake build` in the Mathlib-pre-warmed harness. Returns { success, kind: 'success' | 'parse-error' | 'typecheck-error' | 'build-error' | 'timeout', errors, warnings, sorries }. `sorry` warnings are expected for spec stubs and are surfaced separately from real warnings.",
    {
      source: z.string().describe("Lean 4 source code to typecheck"),
    },
    async ({ source }) => {
      const result = await leanCheck({ source });
      return {
        content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
      };
    }
  );

  server.tool(
    "lean_run",
    "Build + execute a Lean 4 file's `main : IO Unit` entry point. Used by /lean-impl (sub-phase 3b-β) for sanity-checking functional models. Not for spec stubs (which contain `sorry`).",
    {
      source: z.string().describe("Lean 4 source code with a `main : IO Unit` entry point"),
    },
    async ({ source }) => {
      const result = await leanRun({ source });
      return {
        content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
      };
    }
  );

  server.tool(
    "lean_test",
    "Run a Lean 4 test harness (#guard / decide tactics) over a user module. NOTE (3b-α): the runner currently aliases this to `lake build` because `lake test` requires a test driver in lakefile.lean that has not yet been wired; sub-phase 3b-β adds the driver and switches this tool to true `lake test` semantics. Until then, callers expecting test-runner semantics are silently downgraded to compile-time `#guard` checks.",
    {
      source: z.string().describe("Lean 4 source code containing test declarations"),
    },
    async ({ source }) => {
      const result = await leanTest({ source });
      return {
        content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }],
      };
    }
  );

  return server;
}

async function main() {
  const server = createServer();
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});

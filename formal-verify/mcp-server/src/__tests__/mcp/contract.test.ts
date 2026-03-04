import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("../../docker.js", () => ({
  getDockerImage: vi.fn(() => "test-image:latest"),
  runDafny: vi.fn(async () => ({
    exitCode: 0,
    stdout: "Dafny program verifier finished with 1 verified, 0 errors\n",
    stderr: "",
    timedOut: false,
  })),
}));

vi.mock("../../tempdir.js", () => ({
  createTempDir: vi.fn(async () => "/tmp/dafny-test-mock"),
  removeTempDir: vi.fn(async () => {}),
  cleanupStaleDirs: vi.fn(async () => 0),
}));

vi.mock("node:fs/promises", async (importOriginal) => {
  const orig = await importOriginal<typeof import("node:fs/promises")>();
  return {
    ...orig,
    writeFile: vi.fn(async () => {}),
    readdir: vi.fn(async () => []),
    readFile: vi.fn(async () => ""),
  };
});

import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { InMemoryTransport } from "@modelcontextprotocol/sdk/inMemory.js";
import { createServer } from "../../index.js";

describe("MCP Contract", () => {
  let client: Client;

  beforeEach(async () => {
    const server = createServer();
    const [clientTransport, serverTransport] =
      InMemoryTransport.createLinkedPair();
    await server.connect(serverTransport);
    client = new Client({ name: "test-client", version: "1.0.0" });
    await client.connect(clientTransport);
  });

  afterEach(async () => {
    await client.close();
  });

  describe("tool input schemas", () => {
    it("dafny_verify schema has required 'source' string property", async () => {
      const { tools } = await client.listTools();
      const verify = tools.find((t) => t.name === "dafny_verify");

      expect(verify).toBeDefined();
      const schema = verify!.inputSchema;
      expect(schema.type).toBe("object");
      expect(schema.properties).toHaveProperty("source");
      expect((schema.properties as Record<string, { type: string }>).source.type).toBe("string");
      expect(schema.required).toContain("source");
    });

    it("dafny_compile schema has required 'source' string and 'target' enum properties", async () => {
      const { tools } = await client.listTools();
      const compile = tools.find((t) => t.name === "dafny_compile");

      expect(compile).toBeDefined();
      const schema = compile!.inputSchema;
      expect(schema.type).toBe("object");

      const properties = schema.properties as Record<string, { type?: string; enum?: string[] }>;

      // source property
      expect(properties).toHaveProperty("source");
      expect(properties.source.type).toBe("string");

      // target property with enum
      expect(properties).toHaveProperty("target");
      expect(properties.target.enum).toBeDefined();
      expect(properties.target.enum).toContain("py");
      expect(properties.target.enum).toContain("go");
      expect(properties.target.enum).toHaveLength(2);

      // both required
      expect(schema.required).toContain("source");
      expect(schema.required).toContain("target");
    });

    it("dafny_cleanup schema has no required properties", async () => {
      const { tools } = await client.listTools();
      const cleanup = tools.find((t) => t.name === "dafny_cleanup");

      expect(cleanup).toBeDefined();
      const schema = cleanup!.inputSchema;
      expect(schema.type).toBe("object");

      // No required properties (empty or absent)
      const required = schema.required ?? [];
      expect(required).toHaveLength(0);
    });
  });

  describe("response contracts", () => {
    it("dafny_verify response has boolean success, string[] errors, string[] warnings, string rawOutput", async () => {
      const result = await client.callTool({
        name: "dafny_verify",
        arguments: { source: "method Main() {}" },
      });

      const item = (result.content as Array<{ type: string; text: string }>)[0];
      const parsed = JSON.parse(item.text);

      expect(typeof parsed.success).toBe("boolean");
      expect(Array.isArray(parsed.errors)).toBe(true);
      expect(Array.isArray(parsed.warnings)).toBe(true);
      expect(typeof parsed.rawOutput).toBe("string");

      // Verify array element types
      for (const e of parsed.errors) {
        expect(typeof e).toBe("string");
      }
      for (const w of parsed.warnings) {
        expect(typeof w).toBe("string");
      }
    });

    it("dafny_compile response has boolean success, string[] errors, array files, string rawOutput", async () => {
      const result = await client.callTool({
        name: "dafny_compile",
        arguments: { source: "method Main() {}", target: "py" },
      });

      const item = (result.content as Array<{ type: string; text: string }>)[0];
      const parsed = JSON.parse(item.text);

      expect(typeof parsed.success).toBe("boolean");
      expect(Array.isArray(parsed.errors)).toBe(true);
      expect(Array.isArray(parsed.files)).toBe(true);
      expect(typeof parsed.rawOutput).toBe("string");

      // Verify array element types
      for (const e of parsed.errors) {
        expect(typeof e).toBe("string");
      }
      // Each file should have path and content
      for (const f of parsed.files) {
        expect(typeof f.path).toBe("string");
        expect(typeof f.content).toBe("string");
      }
    });

    it("dafny_cleanup response has number cleaned", async () => {
      const result = await client.callTool({
        name: "dafny_cleanup",
        arguments: {},
      });

      const item = (result.content as Array<{ type: string; text: string }>)[0];
      const parsed = JSON.parse(item.text);

      expect(typeof parsed.cleaned).toBe("number");
    });
  });
});

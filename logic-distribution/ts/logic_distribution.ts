#!/usr/bin/env npx tsx
/**
 * JS/TS Codebase Logic Distribution Analyzer
 *
 * Static analysis tool that classifies every function/method in a JavaScript
 * or TypeScript codebase by where its logic lives, answering: "what percentage
 * of code is reachable by formal verification tools?"
 *
 * Usage:
 *   npx tsx logic_distribution.ts /path/to/project [--verbose] [--spot-check N]
 *       [--output results.json] [--summary-only]
 */

import * as ts from "typescript";
import * as fs from "fs";
import * as path from "path";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

enum Category {
  DATABASE_ORM = "DATABASE_ORM",
  SCHEMA_VALIDATION = "SCHEMA_VALIDATION",
  VIEW_FRAMEWORK = "VIEW_FRAMEWORK",
  PURE_FUNCTION = "PURE_FUNCTION",
  EXTERNAL_IO = "EXTERNAL_IO",
  TEST_CODE = "TEST_CODE",
  CONFIGURATION = "CONFIGURATION",
}

enum Confidence {
  HIGH = "HIGH",
  MEDIUM = "MEDIUM",
  LOW = "LOW",
}

interface FunctionInfo {
  filePath: string;
  functionName: string;
  lineNumber: number;
  lineCount: number;
  category: Category;
  confidence: Confidence;
  rationale: string;
  matchedCategories: string[];
  isBorderline: boolean;
  borderlineReason: string;
  className: string | null;
  decorators: string[];
}

interface FileContext {
  filePath: string;
  imports: Set<string>; // module specifiers
  importedNames: Set<string>; // imported identifiers
  frameworkImportedNames: Set<string>; // names from framework modules
  isTestFile: boolean;
  isConfigFile: boolean;
  classBases: Map<string, string[]>; // className -> base class names
}

interface CategoryMatch {
  category: Category;
  rationale: string;
}

// ---------------------------------------------------------------------------
// Detection constants
// ---------------------------------------------------------------------------

// -- DATABASE / ORM --

const ORM_MODULES = new Set([
  "prisma", "@prisma/client",
  "typeorm", "typeorm/",
  "sequelize",
  "knex",
  "drizzle-orm", "drizzle-orm/",
  "mongoose", "mongodb",
  "mikro-orm", "@mikro-orm/",
  "objection", "bookshelf",
  "pg", "mysql", "mysql2", "better-sqlite3", "sqlite3",
  "redis", "ioredis",
  "@supabase/supabase-js",
  "@planetscale/database",
]);

const ORM_METHODS = new Set([
  // Prisma
  "findUnique", "findFirst", "findMany", "create", "createMany",
  "update", "updateMany", "upsert", "delete", "deleteMany",
  "aggregate", "groupBy", "count",
  // Sequelize / TypeORM
  "findAll", "findOne", "findByPk", "findOrCreate", "bulkCreate",
  "findAndCountAll", "getRepository", "createQueryBuilder",
  // Knex / Drizzle
  "select", "insert", "where", "join", "leftJoin", "rightJoin",
  "innerJoin", "orderBy", "groupBy", "having", "limit", "offset",
  "raw", "returning",
  // Mongoose
  "find", "findById", "findOne", "findOneAndUpdate", "findOneAndDelete",
  "populate", "lean", "exec", "save", "aggregate",
  // General
  "query", "execute", "transaction",
]);

const SQL_KEYWORDS = /\b(SELECT\s+.+?\s+FROM|INSERT\s+INTO|UPDATE\s+.+?\s+SET|DELETE\s+FROM|CREATE\s+TABLE|ALTER\s+TABLE)\b/i;

// -- EXTERNAL IO --

const IO_MODULES = new Set([
  "fs", "fs/promises", "node:fs", "node:fs/promises",
  "child_process", "node:child_process",
  "net", "node:net", "dgram", "node:dgram",
  "http", "https", "node:http", "node:https", "http2", "node:http2",
  "nodemailer",
  "axios",
  "node-fetch",
  "got",
  "superagent",
  "ws", "socket.io", "socket.io-client",
  "amqplib", "bull", "bullmq",
  "aws-sdk", "@aws-sdk/",
  "googleapis", "@google-cloud/",
  "nodemailer",
  "@sendgrid/mail",
  "twilio",
  "puppeteer", "playwright",
]);

const IO_GLOBALS = new Set([
  "fetch", "XMLHttpRequest", "WebSocket",
]);

const IO_METHODS = new Set([
  "readFile", "writeFile", "readFileSync", "writeFileSync",
  "readdir", "readdirSync", "mkdir", "mkdirSync", "unlink", "unlinkSync",
  "copyFile", "rename", "stat", "access",
  "exec", "execSync", "spawn", "spawnSync", "fork",
  "createReadStream", "createWriteStream",
  "sendMail", "send",
]);

// -- VIEW / FRAMEWORK --

const FRAMEWORK_MODULES = new Set([
  "react", "react-dom", "react-dom/server", "react-dom/client",
  "next", "next/server", "next/router", "next/navigation", "next/image", "next/link",
  "gatsby",
  "remix", "@remix-run/node", "@remix-run/react", "@remix-run/server-runtime",
  "vue", "nuxt",
  "svelte", "@sveltejs/kit",
  "angular", "@angular/core", "@angular/common", "@angular/router",
  "express",
  "fastify",
  "koa",
  "hono",
  "hapi", "@hapi/hapi",
  "nest", "@nestjs/common", "@nestjs/core",
  "trpc", "@trpc/server", "@trpc/client", "@trpc/react-query",
  "graphql", "apollo-server", "@apollo/server", "graphql-yoga", "type-graphql",
  "@tanstack/react-query", "swr",
  "redux", "@reduxjs/toolkit", "zustand", "jotai", "recoil", "mobx",
  "react-router", "react-router-dom",
  "react-hook-form",
  "formik",
  "@mui/material", "@chakra-ui/react", "antd",
  "tailwindcss",
  "electron",
  "tauri",
]);

const REACT_HOOKS = new Set([
  "useState", "useEffect", "useContext", "useReducer", "useCallback",
  "useMemo", "useRef", "useImperativeHandle", "useLayoutEffect",
  "useDebugValue", "useTransition", "useDeferredValue", "useId",
  "useSyncExternalStore", "useInsertionEffect",
  // React Query / SWR
  "useQuery", "useMutation", "useInfiniteQuery", "useSuspenseQuery",
  "useSWR", "useSWRMutation",
  // Router
  "useRouter", "usePathname", "useSearchParams", "useParams",
  "useNavigate", "useLocation", "useMatch",
  // Redux
  "useSelector", "useDispatch", "useStore",
  // Form
  "useForm", "useFormContext", "useFieldArray",
]);

const EXPRESS_METHODS = new Set([
  "get", "post", "put", "patch", "delete", "use", "all",
  "route", "param", "listen",
]);

const NESTJS_DECORATORS = new Set([
  "Controller", "Get", "Post", "Put", "Patch", "Delete",
  "Injectable", "Module", "Middleware", "Guard", "Interceptor",
  "Pipe", "UseGuards", "UseInterceptors", "UsePipes",
]);

// -- SCHEMA / VALIDATION --

const SCHEMA_MODULES = new Set([
  "zod",
  "yup",
  "joi", "@hapi/joi",
  "class-validator", "class-transformer",
  "superstruct",
  "io-ts",
  "ajv",
  "valibot",
  "typebox", "@sinclair/typebox",
  "arktype",
]);

const SCHEMA_METHODS = new Set([
  // Zod
  "z.object", "z.string", "z.number", "z.boolean", "z.array",
  "z.enum", "z.union", "z.intersection", "z.literal", "z.tuple",
  "z.record", "z.map", "z.set", "z.optional", "z.nullable",
  "z.coerce", "z.preprocess", "z.transform", "z.refine", "z.superRefine",
  "parse", "safeParse", "parseAsync",
  // Yup
  "object", "string", "number", "boolean", "array", "mixed",
  "validate", "validateSync", "isValid",
  // Joi
  "alternatives", "any",
]);

// -- ALL FRAMEWORK MODULES (for pure function detection) --

const ALL_FRAMEWORK_MODULES = new Set([
  ...FRAMEWORK_MODULES,
  ...ORM_MODULES,
  ...IO_MODULES,
  ...SCHEMA_MODULES,
]);

// -- CONFIG FILES --

const CONFIG_PATTERNS = new Set([
  "webpack.config", "rollup.config", "vite.config", "vitest.config",
  "jest.config", "babel.config", "tsconfig",
  "eslint.config", ".eslintrc", "prettier.config", ".prettierrc",
  "postcss.config", "tailwind.config",
  "next.config", "nuxt.config", "svelte.config", "gatsby-config",
  "remix.config",
  "drizzle.config", "knexfile",
  ".env", "env.d",
]);

const EXCLUDED_DIRS = new Set([
  "node_modules", ".next", ".nuxt", ".svelte-kit", ".output",
  "dist", "build", "out", ".cache", ".turbo",
  ".git", ".hg", ".svn",
  "coverage", "__snapshots__",
  ".storybook", "storybook-static",
  "vendor", "third_party",
]);

const FRAMEWORK_FILE_PATTERNS = new Set([
  "middleware", "handler", "handlers", "controller", "controllers",
  "route", "routes", "router", "routers",
  "resolver", "resolvers", "mutation", "mutations",
  "schema", "schemas",
  "serializer", "serializers",
  "guard", "guards", "interceptor", "interceptors",
  "pipe", "pipes", "filter", "filters",
  "gateway", "gateways",
  "module", "provider", "providers",
  "page", "pages", "layout", "layouts",
  "component", "components",
  "hook", "hooks",
  "store", "stores", "slice", "slices",
  "action", "actions", "reducer", "reducers",
  "api",
]);

// ---------------------------------------------------------------------------
// File discovery
// ---------------------------------------------------------------------------

function findSourceFiles(root: string): string[] {
  const files: string[] = [];
  const extensions = new Set([".ts", ".tsx", ".js", ".jsx", ".mjs", ".mts"]);

  function walk(dir: string) {
    let entries: fs.Dirent[];
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }

    for (const entry of entries) {
      if (entry.name.startsWith(".") && entry.isDirectory()) continue;
      if (EXCLUDED_DIRS.has(entry.name) && entry.isDirectory()) continue;

      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(full);
      } else if (entry.isFile()) {
        const ext = path.extname(entry.name);
        if (extensions.has(ext)) {
          // Skip declaration files
          if (entry.name.endsWith(".d.ts") || entry.name.endsWith(".d.mts")) continue;
          files.push(full);
        }
      }
    }
  }

  walk(root);
  return files.sort();
}

// ---------------------------------------------------------------------------
// AST helpers
// ---------------------------------------------------------------------------

function getNodeText(node: ts.Node, sourceFile: ts.SourceFile): string {
  return node.getText(sourceFile);
}

function getLineNumber(node: ts.Node, sourceFile: ts.SourceFile): number {
  return sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1;
}

function getLineCount(node: ts.Node, sourceFile: ts.SourceFile): number {
  const start = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line;
  const end = sourceFile.getLineAndCharacterOfPosition(node.getEnd()).line;
  return end - start + 1;
}

function getDecoratorNames(node: ts.Node): string[] {
  const names: string[] = [];
  const modifiers = ts.canHaveModifiers(node) ? ts.getModifiers(node) : undefined;
  // In TS 5+, decorators are modifiers
  const decorators = ts.canHaveDecorators(node) ? ts.getDecorators(node) : undefined;
  if (decorators) {
    for (const dec of decorators) {
      if (ts.isCallExpression(dec.expression)) {
        names.push(getExpressionName(dec.expression.expression));
      } else {
        names.push(getExpressionName(dec.expression));
      }
    }
  }
  return names;
}

function getExpressionName(expr: ts.Expression): string {
  if (ts.isIdentifier(expr)) {
    return expr.text;
  }
  if (ts.isPropertyAccessExpression(expr)) {
    return getExpressionName(expr.expression) + "." + expr.name.text;
  }
  return "";
}

/** Collect all identifiers and property accesses in a node tree */
function collectReferences(node: ts.Node, sourceFile: ts.SourceFile): {
  identifiers: Set<string>;
  propertyChains: string[];
  callExpressions: string[];
  stringLiterals: string[];
  templateLiterals: string[];
} {
  const identifiers = new Set<string>();
  const propertyChains: string[] = [];
  const callExpressions: string[] = [];
  const stringLiterals: string[] = [];
  const templateLiterals: string[] = [];

  function visit(n: ts.Node) {
    if (ts.isIdentifier(n)) {
      identifiers.add(n.text);
    }

    if (ts.isPropertyAccessExpression(n)) {
      const chain = buildPropertyChain(n);
      if (chain) propertyChains.push(chain);
    }

    if (ts.isCallExpression(n)) {
      const name = buildPropertyChain(n.expression);
      if (name) callExpressions.push(name);
    }

    if (ts.isStringLiteral(n)) {
      stringLiterals.push(n.text);
    }

    if (ts.isNoSubstitutionTemplateLiteral(n)) {
      templateLiterals.push(n.text);
    }

    if (ts.isTemplateExpression(n)) {
      // Collect the head text at minimum
      templateLiterals.push(n.head.text);
    }

    if (ts.isTaggedTemplateExpression(n)) {
      const tag = buildPropertyChain(n.tag);
      if (tag) callExpressions.push(tag);
    }

    ts.forEachChild(n, visit);
  }

  visit(node);
  return { identifiers, propertyChains, callExpressions, stringLiterals, templateLiterals };
}

function buildPropertyChain(expr: ts.Node): string {
  if (ts.isIdentifier(expr)) {
    return expr.text;
  }
  if (ts.isPropertyAccessExpression(expr)) {
    const left = buildPropertyChain(expr.expression);
    return left ? `${left}.${expr.name.text}` : expr.name.text;
  }
  if (ts.isCallExpression(expr)) {
    return buildPropertyChain(expr.expression);
  }
  if (ts.isElementAccessExpression(expr)) {
    return buildPropertyChain(expr.expression);
  }
  if (ts.isParenthesizedExpression(expr)) {
    return buildPropertyChain(expr.expression);
  }
  if (ts.isNonNullExpression(expr)) {
    return buildPropertyChain(expr.expression);
  }
  if (ts.isAsExpression(expr)) {
    return buildPropertyChain(expr.expression);
  }
  // this.x
  if (ts.isThisKeyword && expr.kind === ts.SyntaxKind.ThisKeyword) {
    return "this";
  }
  return "";
}

function getBaseClassNames(node: ts.ClassDeclaration | ts.ClassExpression): string[] {
  const bases: string[] = [];
  if (node.heritageClauses) {
    for (const clause of node.heritageClauses) {
      if (clause.token === ts.SyntaxKind.ExtendsKeyword) {
        for (const type of clause.types) {
          const name = buildPropertyChain(type.expression);
          if (name) bases.push(name);
        }
      }
    }
  }
  return bases;
}

// ---------------------------------------------------------------------------
// File context analysis
// ---------------------------------------------------------------------------

function analyzeFileContext(filePath: string, sourceFile: ts.SourceFile): FileContext {
  const ctx: FileContext = {
    filePath,
    imports: new Set(),
    importedNames: new Set(),
    frameworkImportedNames: new Set(),
    isTestFile: false,
    isConfigFile: false,
    classBases: new Map(),
  };

  const basename = path.basename(filePath);
  const basenameNoExt = basename.replace(/\.(ts|tsx|js|jsx|mjs|mts)$/, "");
  const dirParts = filePath.split(path.sep);

  // Test file detection
  ctx.isTestFile = (
    basename.includes(".test.") ||
    basename.includes(".spec.") ||
    basename.startsWith("test.") ||
    basename === "jest.setup.ts" ||
    basename === "jest.setup.js" ||
    basename.includes("__tests__") ||
    dirParts.includes("__tests__") ||
    dirParts.includes("test") ||
    dirParts.includes("tests") ||
    dirParts.includes("__mocks__") ||
    basename.endsWith(".stories.tsx") ||
    basename.endsWith(".stories.ts") ||
    basename.endsWith(".stories.jsx") ||
    basename.endsWith(".stories.js")
  );

  // Config file detection
  ctx.isConfigFile = (
    [...CONFIG_PATTERNS].some(p => basenameNoExt.startsWith(p) || basenameNoExt === p) ||
    basename === "package.json" ||
    basename.startsWith(".") ||
    dirParts.includes("migrations") ||
    dirParts.includes("seeds") ||
    dirParts.includes("fixtures")
  );

  // Analyze imports
  ts.forEachChild(sourceFile, (node) => {
    if (ts.isImportDeclaration(node) && ts.isStringLiteral(node.moduleSpecifier)) {
      const mod = node.moduleSpecifier.text;
      ctx.imports.add(mod);

      // Extract root module (e.g. "@prisma/client" -> "@prisma/client", "react" -> "react")
      const rootMod = mod.startsWith("@") ? mod.split("/").slice(0, 2).join("/") : mod.split("/")[0];
      ctx.imports.add(rootMod);

      const isFramework = ALL_FRAMEWORK_MODULES.has(rootMod) || ALL_FRAMEWORK_MODULES.has(mod) ||
        [...ALL_FRAMEWORK_MODULES].some(fm => mod.startsWith(fm));

      if (node.importClause) {
        // Default import
        if (node.importClause.name) {
          const name = node.importClause.name.text;
          ctx.importedNames.add(name);
          if (isFramework) ctx.frameworkImportedNames.add(name);
        }
        // Named imports
        if (node.importClause.namedBindings) {
          if (ts.isNamedImports(node.importClause.namedBindings)) {
            for (const spec of node.importClause.namedBindings.elements) {
              const name = spec.name.text;
              ctx.importedNames.add(name);
              if (isFramework) ctx.frameworkImportedNames.add(name);
            }
          }
          // Namespace import
          if (ts.isNamespaceImport(node.importClause.namedBindings)) {
            const name = node.importClause.namedBindings.name.text;
            ctx.importedNames.add(name);
            if (isFramework) ctx.frameworkImportedNames.add(name);
          }
        }
      }
    }

    // Dynamic imports: require("...")
    if (ts.isVariableStatement(node)) {
      ts.forEachChild(node, (decl) => {
        if (ts.isVariableDeclarationList(decl)) {
          for (const d of decl.declarations) {
            if (d.initializer && ts.isCallExpression(d.initializer)) {
              const fn = d.initializer.expression;
              if (ts.isIdentifier(fn) && fn.text === "require" && d.initializer.arguments.length > 0) {
                const arg = d.initializer.arguments[0];
                if (ts.isStringLiteral(arg)) {
                  ctx.imports.add(arg.text);
                }
              }
            }
          }
        }
      });
    }

    // Class hierarchy
    if (ts.isClassDeclaration(node) && node.name) {
      ctx.classBases.set(node.name.text, getBaseClassNames(node));
    }
  });

  return ctx;
}

// ---------------------------------------------------------------------------
// Function classifier
// ---------------------------------------------------------------------------

function classifyFunction(
  node: ts.Node,
  ctx: FileContext,
  sourceFile: ts.SourceFile,
  enclosingClassName: string | null,
): FunctionInfo {
  let funcName = "<anonymous>";
  let className = enclosingClassName;

  // Determine function name
  if (ts.isFunctionDeclaration(node) && node.name) {
    funcName = node.name.text;
  } else if (ts.isMethodDeclaration(node) && ts.isIdentifier(node.name)) {
    funcName = node.name.text;
  } else if (ts.isGetAccessorDeclaration(node) || ts.isSetAccessorDeclaration(node)) {
    if (ts.isIdentifier(node.name)) funcName = node.name.text;
  } else if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name)) {
    funcName = node.name.text;
  } else if (ts.isPropertyDeclaration(node) && ts.isIdentifier(node.name)) {
    funcName = node.name.text;
  } else if (ts.isPropertyAssignment(node) && ts.isIdentifier(node.name)) {
    funcName = node.name.text;
  } else if (ts.isExportAssignment(node)) {
    funcName = "default export";
  }

  const fullName = className ? `${className}.${funcName}` : funcName;
  const lineNumber = getLineNumber(node, sourceFile);
  const lineCount = getLineCount(node, sourceFile);
  const decorators = getDecoratorNames(node);

  // -- Priority 1: TEST_CODE --
  if (ctx.isTestFile) {
    return {
      filePath: ctx.filePath, functionName: fullName,
      lineNumber, lineCount,
      category: Category.TEST_CODE, confidence: Confidence.HIGH,
      rationale: "File is a test file",
      matchedCategories: [Category.TEST_CODE],
      isBorderline: false, borderlineReason: "",
      className, decorators,
    };
  }

  // -- Priority 2: CONFIGURATION --
  if (ctx.isConfigFile) {
    return {
      filePath: ctx.filePath, functionName: fullName,
      lineNumber, lineCount,
      category: Category.CONFIGURATION, confidence: Confidence.HIGH,
      rationale: "File is a configuration file",
      matchedCategories: [Category.CONFIGURATION],
      isBorderline: false, borderlineReason: "",
      className, decorators,
    };
  }

  // Gather references from function body
  const refs = collectReferences(node, sourceFile);
  const { identifiers, propertyChains, callExpressions, stringLiterals, templateLiterals } = refs;

  const allStrings = [...stringLiterals, ...templateLiterals].join(" ");
  const matched: CategoryMatch[] = [];

  // -- Check DATABASE_ORM --
  const ormReasons: string[] = [];

  // ORM module usage
  for (const mod of ctx.imports) {
    const rootMod = mod.startsWith("@") ? mod.split("/").slice(0, 2).join("/") : mod.split("/")[0];
    if (ORM_MODULES.has(rootMod) || ORM_MODULES.has(mod)) {
      // Check if this function actually uses ORM methods
      for (const call of callExpressions) {
        const parts = call.split(".");
        const lastPart = parts[parts.length - 1];
        if (ORM_METHODS.has(lastPart)) {
          ormReasons.push(`ORM call: ${call}`);
          break;
        }
      }
      // Check for prisma-style chains: prisma.user.findMany
      for (const chain of propertyChains) {
        if (chain.startsWith("prisma.") || chain.startsWith("db.") ||
            chain.includes(".prisma.")) {
          const parts = chain.split(".");
          if (parts.some(p => ORM_METHODS.has(p))) {
            ormReasons.push(`Prisma/DB chain: ${chain}`);
          }
        }
      }
    }
  }

  // Direct ORM method calls on known patterns
  for (const call of callExpressions) {
    // model.findMany(), Model.find(), etc.
    const parts = call.split(".");
    const last = parts[parts.length - 1];
    if (ORM_METHODS.has(last) && parts.length >= 2) {
      // Verify it's not a generic .find() on an array
      const ctx_name = parts[parts.length - 2];
      if (ctx_name !== "Array" && ctx_name !== "array" &&
          !["filter", "map", "reduce", "forEach"].includes(last)) {
        // Check for common ORM patterns
        if (["findUnique", "findFirst", "findMany", "createMany", "updateMany",
             "deleteMany", "findAll", "findByPk", "findAndCountAll", "findOrCreate",
             "bulkCreate", "createQueryBuilder", "getRepository",
             "findById", "findOneAndUpdate", "findOneAndDelete",
             "populate", "lean", "aggregate",
             "select", "insert", "where", "join", "leftJoin", "rightJoin",
             "innerJoin", "having", "raw", "returning",
             "transaction", "execute",
        ].includes(last)) {
          ormReasons.push(`ORM method: ${call}`);
        }
      }
    }
  }

  // SQL in template literals (tagged or regular)
  for (const call of callExpressions) {
    if (call === "sql" || call.endsWith(".sql") || call === "Prisma.sql" || call === "knex.raw") {
      ormReasons.push(`SQL template: ${call}`);
    }
  }
  if (SQL_KEYWORDS.test(allStrings)) {
    ormReasons.push("SQL in string literal");
  }

  if (ormReasons.length > 0) {
    matched.push({ category: Category.DATABASE_ORM, rationale: ormReasons.slice(0, 3).join("; ") });
  }

  // -- Check EXTERNAL_IO --
  const ioReasons: string[] = [];

  for (const mod of ctx.imports) {
    const rootMod = mod.startsWith("@") ? mod.split("/").slice(0, 2).join("/") : mod.split("/")[0];
    if (IO_MODULES.has(rootMod) || IO_MODULES.has(mod)) {
      for (const call of callExpressions) {
        const parts = call.split(".");
        const last = parts[parts.length - 1];
        if (IO_METHODS.has(last) || parts[0] === "fs" || parts[0] === "child_process") {
          ioReasons.push(`IO call: ${call}`);
          break;
        }
      }
      // Check for http/axios usage
      for (const call of callExpressions) {
        if (call.startsWith("axios.") || call.startsWith("got.") ||
            call.startsWith("superagent.")) {
          ioReasons.push(`HTTP client: ${call}`);
          break;
        }
      }
    }
  }

  // fetch() calls
  if (identifiers.has("fetch") || callExpressions.includes("fetch")) {
    ioReasons.push("fetch() call");
  }

  // XMLHttpRequest
  if (identifiers.has("XMLHttpRequest")) {
    ioReasons.push("XMLHttpRequest usage");
  }

  // fs operations
  for (const call of callExpressions) {
    const parts = call.split(".");
    const last = parts[parts.length - 1];
    if (IO_METHODS.has(last) && (parts.includes("fs") || parts.includes("promises"))) {
      ioReasons.push(`File IO: ${call}`);
    }
  }

  // process.env access patterns with writes
  for (const call of callExpressions) {
    if (call.startsWith("process.") && ["exit", "kill", "send"].some(m => call.includes(m))) {
      ioReasons.push(`Process: ${call}`);
    }
  }

  // WebSocket
  if (identifiers.has("WebSocket") || identifiers.has("ws")) {
    for (const call of callExpressions) {
      if (call.includes("WebSocket") || call.includes("ws.")) {
        ioReasons.push(`WebSocket: ${call}`);
        break;
      }
    }
  }

  if (ioReasons.length > 0) {
    matched.push({ category: Category.EXTERNAL_IO, rationale: ioReasons.slice(0, 3).join("; ") });
  }

  // -- Check VIEW_FRAMEWORK --
  const viewReasons: string[] = [];

  // React: JSX detection (any function returning JSX is a component/view)
  if (containsJSX(node)) {
    viewReasons.push("Returns JSX (React/UI component)");
  }

  // React hooks
  for (const call of callExpressions) {
    const parts = call.split(".");
    const fn = parts[parts.length - 1];
    if (REACT_HOOKS.has(fn)) {
      viewReasons.push(`React hook: ${fn}`);
      break;
    }
    // Custom hooks (useXxx)
    if (fn.startsWith("use") && fn.length > 3 && fn[3] === fn[3].toUpperCase()) {
      if (ctx.imports.has("react") || identifiers.has("useState") || identifiers.has("useEffect")) {
        viewReasons.push(`Custom hook: ${fn}`);
        break;
      }
    }
  }

  // Function name starts with "use" and is in a React context
  if (funcName.startsWith("use") && funcName.length > 3 && funcName[3] === funcName[3].toUpperCase()) {
    if (ctx.imports.has("react") || ctx.importedNames.has("useState") || ctx.importedNames.has("useEffect")) {
      viewReasons.push(`Custom hook definition: ${funcName}`);
    }
  }

  // Express/Fastify/Koa route handlers
  for (const call of callExpressions) {
    const parts = call.split(".");
    const last = parts[parts.length - 1];
    if (EXPRESS_METHODS.has(last) && parts.length >= 2) {
      const obj = parts[parts.length - 2];
      if (["app", "router", "server", "api", "route"].includes(obj)) {
        viewReasons.push(`Route handler: ${call}`);
        break;
      }
    }
  }

  // Middleware pattern: (req, res, next) or (req, res) or (ctx, next)
  if (ts.isFunctionDeclaration(node) || ts.isArrowFunction(node) || ts.isFunctionExpression(node)) {
    const params = node.parameters;
    if (params.length >= 2 && params.length <= 3) {
      const paramNames = params.map(p => ts.isIdentifier(p.name) ? p.name.text : "");
      if ((paramNames.includes("req") && paramNames.includes("res")) ||
          (paramNames.includes("request") && paramNames.includes("response")) ||
          (paramNames.includes("ctx") && paramNames.includes("next"))) {
        viewReasons.push(`Middleware/handler signature: (${paramNames.join(", ")})`);
      }
    }
  }

  // NestJS decorators
  for (const dec of decorators) {
    if (NESTJS_DECORATORS.has(dec)) {
      viewReasons.push(`NestJS decorator: @${dec}`);
    }
  }

  // GraphQL resolvers
  if (enclosingClassName) {
    const bases = ctx.classBases.get(enclosingClassName) || [];
    if (bases.some(b => b.includes("Resolver") || b.includes("Controller"))) {
      viewReasons.push(`Resolver/Controller method: ${fullName}`);
    }
  }

  // tRPC procedures
  for (const chain of propertyChains) {
    if (chain.includes("publicProcedure") || chain.includes("protectedProcedure") ||
        chain.includes("procedure") || chain.includes("router")) {
      if (ctx.imports.has("@trpc/server") || ctx.imports.has("trpc")) {
        viewReasons.push(`tRPC procedure: ${chain}`);
        break;
      }
    }
  }

  // Next.js specific patterns
  if (funcName === "getServerSideProps" || funcName === "getStaticProps" ||
      funcName === "getStaticPaths" || funcName === "generateMetadata" ||
      funcName === "generateStaticParams") {
    viewReasons.push(`Next.js data function: ${funcName}`);
  }

  // Remix loaders/actions
  if (funcName === "loader" || funcName === "action") {
    if (ctx.imports.has("@remix-run/node") || ctx.imports.has("@remix-run/react") || ctx.imports.has("remix")) {
      viewReasons.push(`Remix ${funcName}`);
    }
  }

  // Redux/state management
  for (const call of callExpressions) {
    if (call.includes("createSlice") || call.includes("createAsyncThunk") ||
        call.includes("configureStore") || call.includes("createStore") ||
        call.includes("createReducer") || call.includes("createAction")) {
      viewReasons.push(`State management: ${call}`);
      break;
    }
  }

  // Functions in framework-layer files that use framework imports
  const fileBasename = path.basename(ctx.filePath).replace(/\.(ts|tsx|js|jsx|mjs|mts)$/, "");
  const fileParts = new Set(ctx.filePath.split(path.sep));
  const isFrameworkFile = FRAMEWORK_FILE_PATTERNS.has(fileBasename) ||
    [...fileParts].some(p => FRAMEWORK_FILE_PATTERNS.has(p));

  if (isFrameworkFile && viewReasons.length === 0 && matched.length === 0) {
    const usesFramework = [...ctx.frameworkImportedNames].some(n => identifiers.has(n));
    if (usesFramework) {
      viewReasons.push(`Function in framework file: ${fileBasename}`);
    }
  }

  if (viewReasons.length > 0) {
    matched.push({ category: Category.VIEW_FRAMEWORK, rationale: viewReasons.slice(0, 3).join("; ") });
  }

  // -- Check SCHEMA_VALIDATION --
  const schemaReasons: string[] = [];

  for (const mod of ctx.imports) {
    if (SCHEMA_MODULES.has(mod)) {
      // Check for schema definition patterns
      for (const call of callExpressions) {
        if (call.startsWith("z.") || call.startsWith("yup.") || call.startsWith("Joi.") ||
            call.startsWith("Type.") || call.startsWith("t.") ||
            SCHEMA_METHODS.has(call.split(".").pop() || "")) {
          schemaReasons.push(`Schema definition: ${call}`);
          break;
        }
      }
      for (const id of identifiers) {
        if (id === "z" || id === "yup" || id === "Joi" || id === "schema" || id === "Schema") {
          schemaReasons.push(`Schema library: ${id}`);
          break;
        }
      }
    }
  }

  // class-validator decorators
  for (const dec of decorators) {
    if (["IsString", "IsNumber", "IsBoolean", "IsEmail", "IsOptional",
         "IsNotEmpty", "MinLength", "MaxLength", "Min", "Max",
         "ValidateNested", "IsArray", "IsEnum", "IsDate", "IsUrl",
    ].includes(dec)) {
      schemaReasons.push(`Validation decorator: @${dec}`);
    }
  }

  if (schemaReasons.length > 0) {
    matched.push({ category: Category.SCHEMA_VALIDATION, rationale: schemaReasons.slice(0, 3).join("; ") });
  }

  // -- Check PURE_FUNCTION (by absence) --
  const usesFrameworkNames = [...ctx.frameworkImportedNames].some(n => identifiers.has(n));

  // Side effects
  const hasConsoleLog = callExpressions.some(c => c.startsWith("console."));
  const hasThrow = containsThrow(node);

  // -- Apply priority --
  const allMatchedCats = matched.map(m => m.category);

  if (matched.length === 0) {
    // If uses framework-imported names, classify as VIEW_FRAMEWORK
    if (usesFrameworkNames) {
      return {
        filePath: ctx.filePath, functionName: fullName,
        lineNumber, lineCount,
        category: Category.VIEW_FRAMEWORK, confidence: Confidence.LOW,
        rationale: `Uses framework-imported names`,
        matchedCategories: [Category.VIEW_FRAMEWORK],
        isBorderline: true,
        borderlineReason: "Classified as framework code due to imported name usage",
        className, decorators,
      };
    }

    // Pure function
    const hasFrameworkImports = ctx.imports.size > 0 &&
      [...ctx.imports].some(m => {
        const root = m.startsWith("@") ? m.split("/").slice(0, 2).join("/") : m.split("/")[0];
        return ALL_FRAMEWORK_MODULES.has(root) || ALL_FRAMEWORK_MODULES.has(m);
      });

    let confidence: Confidence;
    let rationale: string;
    let isBorderline = false;
    let borderlineReason = "";

    if (hasConsoleLog) {
      confidence = Confidence.MEDIUM;
      rationale = "Pure function (has console.log side effects)";
      isBorderline = true;
      borderlineReason = "Uses console.log";
    } else if (hasFrameworkImports) {
      confidence = Confidence.LOW;
      rationale = "No framework/IO indicators detected in function body";
    } else {
      confidence = Confidence.HIGH;
      rationale = "No framework/IO/ORM indicators detected";
    }

    return {
      filePath: ctx.filePath, functionName: fullName,
      lineNumber, lineCount,
      category: Category.PURE_FUNCTION, confidence,
      rationale,
      matchedCategories: allMatchedCats.length > 0 ? allMatchedCats : [Category.PURE_FUNCTION],
      isBorderline, borderlineReason,
      className, decorators,
    };
  }

  // Priority: DATABASE_ORM > EXTERNAL_IO > VIEW_FRAMEWORK > SCHEMA_VALIDATION
  const priorityOrder = [
    Category.DATABASE_ORM,
    Category.EXTERNAL_IO,
    Category.VIEW_FRAMEWORK,
    Category.SCHEMA_VALIDATION,
  ];

  let chosen: CategoryMatch | null = null;
  for (const cat of priorityOrder) {
    const m = matched.find(m => m.category === cat);
    if (m) { chosen = m; break; }
  }
  if (!chosen) chosen = matched[0];

  const confidence = matched.length === 1 ? Confidence.HIGH : Confidence.MEDIUM;
  const isBorderline = matched.length > 1;
  const borderlineReason = isBorderline
    ? `Also matched: ${matched.filter(m => m.category !== chosen!.category).map(m => m.category).join(", ")}`
    : "";

  return {
    filePath: ctx.filePath, functionName: fullName,
    lineNumber, lineCount,
    category: chosen.category, confidence,
    rationale: chosen.rationale,
    matchedCategories: allMatchedCats,
    isBorderline, borderlineReason,
    className, decorators,
  };
}

function containsJSX(node: ts.Node): boolean {
  let found = false;
  function visit(n: ts.Node) {
    if (found) return;
    if (ts.isJsxElement(n) || ts.isJsxSelfClosingElement(n) || ts.isJsxFragment(n)) {
      found = true;
      return;
    }
    ts.forEachChild(n, visit);
  }
  ts.forEachChild(node, visit);
  return found;
}

function containsThrow(node: ts.Node): boolean {
  let found = false;
  function visit(n: ts.Node) {
    if (found) return;
    if (ts.isThrowStatement(n)) { found = true; return; }
    ts.forEachChild(n, visit);
  }
  ts.forEachChild(node, visit);
  return found;
}

// ---------------------------------------------------------------------------
// File analysis
// ---------------------------------------------------------------------------

function analyzeFile(filePath: string, verbose: boolean): FunctionInfo[] {
  let source: string;
  try {
    source = fs.readFileSync(filePath, "utf-8");
  } catch {
    if (verbose) console.error(`  [SKIP] Cannot read ${filePath}`);
    return [];
  }

  const ext = path.extname(filePath);
  const scriptKind = [".tsx", ".jsx"].includes(ext) ? ts.ScriptKind.TSX :
    [".ts", ".mts"].includes(ext) ? ts.ScriptKind.TS : ts.ScriptKind.JS;

  const sourceFile = ts.createSourceFile(filePath, source, ts.ScriptTarget.Latest, true, scriptKind);
  const ctx = analyzeFileContext(filePath, sourceFile);
  const results: FunctionInfo[] = [];

  function extractFunctionNode(node: ts.Node): ts.Node | null {
    // Arrow function or function expression in variable declaration
    if (ts.isVariableDeclaration(node) && node.initializer) {
      if (ts.isArrowFunction(node.initializer) || ts.isFunctionExpression(node.initializer)) {
        return node;
      }
      // const Component = React.memo(() => ...) or similar wrappers
      if (ts.isCallExpression(node.initializer)) {
        const args = node.initializer.arguments;
        if (args.length > 0 && (ts.isArrowFunction(args[0]) || ts.isFunctionExpression(args[0]))) {
          return node;
        }
      }
    }
    return null;
  }

  function visitTopLevel(node: ts.Node) {
    // Function declarations
    if (ts.isFunctionDeclaration(node) && node.name) {
      const info = classifyFunction(node, ctx, sourceFile, null);
      results.push(info);
      if (verbose) printClassification(info);
      return;
    }

    // Variable declarations with arrow/function expressions
    if (ts.isVariableStatement(node)) {
      for (const decl of node.declarationList.declarations) {
        const fn = extractFunctionNode(decl);
        if (fn) {
          const info = classifyFunction(fn, ctx, sourceFile, null);
          results.push(info);
          if (verbose) printClassification(info);
        }
      }
      return;
    }

    // Export default function
    if (ts.isExportAssignment(node)) {
      if (node.expression && (ts.isArrowFunction(node.expression) || ts.isFunctionExpression(node.expression))) {
        const info = classifyFunction(node, ctx, sourceFile, null);
        results.push(info);
        if (verbose) printClassification(info);
      }
      return;
    }

    // Class declarations
    if (ts.isClassDeclaration(node) && node.name) {
      const className = node.name.text;
      for (const member of node.members) {
        if (ts.isMethodDeclaration(member) || ts.isGetAccessorDeclaration(member) ||
            ts.isSetAccessorDeclaration(member)) {
          const info = classifyFunction(member, ctx, sourceFile, className);
          results.push(info);
          if (verbose) printClassification(info);
        }
        // Property with arrow function
        if (ts.isPropertyDeclaration(member) && member.initializer) {
          if (ts.isArrowFunction(member.initializer) || ts.isFunctionExpression(member.initializer)) {
            const info = classifyFunction(member, ctx, sourceFile, className);
            results.push(info);
            if (verbose) printClassification(info);
          }
        }
      }
      return;
    }

    // Export declarations wrapping functions
    if (ts.isExportDeclaration(node)) return;

    // Handle `export function ...` and `export const ...`
    if (ts.isExportAssignment(node)) return;
  }

  ts.forEachChild(sourceFile, visitTopLevel);
  return results;
}

function printClassification(info: FunctionInfo) {
  console.error(`  ${info.category.padEnd(20)} ${info.functionName} ` +
    `(L${info.lineNumber}, ${info.lineCount} lines) ` +
    `[${info.confidence}] ${info.rationale}`);
}

// ---------------------------------------------------------------------------
// Project analysis
// ---------------------------------------------------------------------------

function analyzeProject(root: string, verbose: boolean): FunctionInfo[] {
  const files = findSourceFiles(root);
  if (verbose) console.error(`Found ${files.length} source files to analyze`);

  const results: FunctionInfo[] = [];
  for (const f of files) {
    if (verbose) {
      const rel = path.relative(root, f);
      console.error(`\nAnalyzing: ${rel}`);
    }
    results.push(...analyzeFile(f, verbose));
  }
  return results;
}

// ---------------------------------------------------------------------------
// Reporting
// ---------------------------------------------------------------------------

const MAIN_CATEGORIES = [
  Category.DATABASE_ORM,
  Category.SCHEMA_VALIDATION,
  Category.VIEW_FRAMEWORK,
  Category.PURE_FUNCTION,
  Category.EXTERNAL_IO,
];

function printSummary(results: FunctionInfo[], projectRoot: string) {
  const main = results.filter(r => MAIN_CATEGORIES.includes(r.category));
  const tests = results.filter(r => r.category === Category.TEST_CODE);
  const config = results.filter(r => r.category === Category.CONFIGURATION);

  const catFuncs = new Map<string, number>();
  const catLines = new Map<string, number>();
  for (const r of main) {
    catFuncs.set(r.category, (catFuncs.get(r.category) || 0) + 1);
    catLines.set(r.category, (catLines.get(r.category) || 0) + r.lineCount);
  }

  const totalFuncs = [...catFuncs.values()].reduce((a, b) => a + b, 0);
  const totalLines = [...catLines.values()].reduce((a, b) => a + b, 0);

  console.log("");
  console.log("=".repeat(80));
  console.log("LOGIC DISTRIBUTION ANALYSIS (JS/TS)");
  console.log("=".repeat(80));
  console.log(`Project: ${projectRoot}`);
  console.log(`Total functions analyzed: ${results.length}`);
  console.log("");

  const header = `${"Category".padEnd(22)} ${"Functions".padStart(10)} ${"Lines of Code".padStart(15)} ${"% Functions".padStart(12)} ${"% Lines".padStart(10)}`;
  console.log(header);
  console.log("-".repeat(header.length));

  for (const cat of MAIN_CATEGORIES) {
    const funcs = catFuncs.get(cat) || 0;
    const lines = catLines.get(cat) || 0;
    const pctFuncs = totalFuncs ? (funcs / totalFuncs * 100).toFixed(1) : "0.0";
    const pctLines = totalLines ? (lines / totalLines * 100).toFixed(1) : "0.0";
    console.log(
      `${cat.padEnd(22)} ${funcs.toLocaleString().padStart(10)} ${lines.toLocaleString().padStart(15)} ${(pctFuncs + "%").padStart(12)} ${(pctLines + "%").padStart(10)}`
    );
  }

  console.log("-".repeat(header.length));
  console.log(
    `${"TOTAL".padEnd(22)} ${totalFuncs.toLocaleString().padStart(10)} ${totalLines.toLocaleString().padStart(15)} ${"100.0%".padStart(12)} ${"100.0%".padStart(10)}`
  );
  console.log("");
  console.log(`Excluded: ${tests.length} test functions, ${config.length} configuration functions`);
}

function printPureFunctions(results: FunctionInfo[], projectRoot: string) {
  const pure = results.filter(r => r.category === Category.PURE_FUNCTION);
  if (pure.length === 0) {
    console.log("\nNo pure functions found.");
    return;
  }

  console.log("");
  console.log("=".repeat(80));
  console.log(`PURE FUNCTIONS (formally verifiable) — ${pure.length} functions`);
  console.log("=".repeat(80));

  pure.sort((a, b) => a.filePath.localeCompare(b.filePath) || a.lineNumber - b.lineNumber);
  for (const r of pure.slice(0, 100)) {
    const rel = path.relative(projectRoot, r.filePath);
    const borderline = r.isBorderline ? " *BORDERLINE*" : "";
    console.log(`  ${rel}:${r.lineNumber}  ${r.functionName} (${r.lineCount} lines) [${r.confidence}]${borderline}`);
    if (r.borderlineReason) console.log(`    Note: ${r.borderlineReason}`);
  }
  if (pure.length > 100) {
    console.log(`  ... and ${pure.length - 100} more`);
  }
}

function printBorderlineCases(results: FunctionInfo[], projectRoot: string) {
  const borderline = results.filter(r => r.isBorderline && MAIN_CATEGORIES.includes(r.category));
  if (borderline.length === 0) {
    console.log("\nNo borderline cases found.");
    return;
  }

  console.log("");
  console.log("=".repeat(80));
  console.log(`BORDERLINE CASES — ${borderline.length} functions`);
  console.log("=".repeat(80));

  borderline.sort((a, b) => a.filePath.localeCompare(b.filePath) || a.lineNumber - b.lineNumber);
  for (const r of borderline.slice(0, 50)) {
    const rel = path.relative(projectRoot, r.filePath);
    console.log(`  ${rel}:${r.lineNumber}  ${r.functionName} → ${r.category}`);
    console.log(`    Reason: ${r.borderlineReason}`);
    console.log(`    Rationale: ${r.rationale}`);
  }
  if (borderline.length > 50) {
    console.log(`  ... and ${borderline.length - 50} more`);
  }
}

function printPerFileBreakdown(results: FunctionInfo[], projectRoot: string) {
  console.log("");
  console.log("=".repeat(80));
  console.log("PER-FILE BREAKDOWN");
  console.log("=".repeat(80));

  const byFile = new Map<string, FunctionInfo[]>();
  for (const r of results) {
    const list = byFile.get(r.filePath) || [];
    list.push(r);
    byFile.set(r.filePath, list);
  }

  const sortedFiles = [...byFile.keys()].sort();
  for (const fp of sortedFiles) {
    const funcs = byFile.get(fp)!;
    const rel = path.relative(projectRoot, fp);
    const counts = new Map<string, number>();
    for (const f of funcs) {
      counts.set(f.category, (counts.get(f.category) || 0) + 1);
    }
    const parts = [...counts.entries()].map(([cat, count]) => `${cat}: ${count}`).join(" | ");
    console.log(`  ${rel} (${funcs.length} funcs): ${parts}`);
  }
}

function printSpotCheck(results: FunctionInfo[], n: number, projectRoot: string) {
  const main = results.filter(r => MAIN_CATEGORIES.includes(r.category));
  const sample: FunctionInfo[] = [];
  const indices = new Set<number>();
  const count = Math.min(n, main.length);
  while (indices.size < count) {
    indices.add(Math.floor(Math.random() * main.length));
  }
  for (const i of indices) sample.push(main[i]);

  console.log("");
  console.log("=".repeat(80));
  console.log(`SPOT CHECK — ${sample.length} randomly sampled functions`);
  console.log("=".repeat(80));

  for (const r of sample) {
    const rel = path.relative(projectRoot, r.filePath);
    console.log(`\n  ${rel}:${r.lineNumber}`);
    console.log(`  Function: ${r.functionName}`);
    console.log(`  Category: ${r.category} [${r.confidence}]`);
    console.log(`  Lines: ${r.lineCount}`);
    console.log(`  Rationale: ${r.rationale}`);
    if (r.isBorderline) console.log(`  Borderline: ${r.borderlineReason}`);
    if (r.decorators.length) console.log(`  Decorators: ${r.decorators.join(", ")}`);
  }
}

function writeJsonOutput(results: FunctionInfo[], outputPath: string, projectRoot: string) {
  const main = results.filter(r => MAIN_CATEGORIES.includes(r.category));

  const catFuncs = new Map<string, number>();
  const catLines = new Map<string, number>();
  for (const r of main) {
    catFuncs.set(r.category, (catFuncs.get(r.category) || 0) + 1);
    catLines.set(r.category, (catLines.get(r.category) || 0) + r.lineCount);
  }

  const totalFuncs = [...catFuncs.values()].reduce((a, b) => a + b, 0);
  const totalLines = [...catLines.values()].reduce((a, b) => a + b, 0);

  const categories: Record<string, any> = {};
  for (const cat of MAIN_CATEGORIES) {
    const funcs = catFuncs.get(cat) || 0;
    const lines = catLines.get(cat) || 0;
    categories[cat] = {
      functions: funcs,
      lines,
      pctFunctions: totalFuncs ? +(funcs / totalFuncs * 100).toFixed(1) : 0,
      pctLines: totalLines ? +(lines / totalLines * 100).toFixed(1) : 0,
    };
  }

  const data = {
    projectRoot,
    totalFunctions: results.length,
    summary: {
      mainFunctions: totalFuncs,
      mainLines: totalLines,
      testFunctions: results.filter(r => r.category === Category.TEST_CODE).length,
      configFunctions: results.filter(r => r.category === Category.CONFIGURATION).length,
      categories,
    },
    sampleFunctions: results.slice(0, 50).map(r => ({
      filePath: path.relative(projectRoot, r.filePath),
      functionName: r.functionName,
      lineNumber: r.lineNumber,
      lineCount: r.lineCount,
      category: r.category,
      confidence: r.confidence,
      rationale: r.rationale,
      matchedCategories: r.matchedCategories,
      isBorderline: r.isBorderline,
      borderlineReason: r.borderlineReason,
      className: r.className,
      decorators: r.decorators,
    })),
  };

  fs.writeFileSync(outputPath, JSON.stringify(data, null, 2), "utf-8");
  console.log(`\nJSON output written to: ${outputPath}`);
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

function main() {
  const args = process.argv.slice(2);

  if (args.length === 0 || args.includes("--help") || args.includes("-h")) {
    console.log(`Usage: npx tsx logic_distribution.ts <project-root> [options]

Options:
  --verbose           Print each classification decision
  --spot-check N      Randomly sample N functions for review
  --output PATH       JSON output path (default: analysis_results.json)
  --no-json           Skip JSON output
  --summary-only      Only print summary table

Examples:
  npx tsx logic_distribution.ts /path/to/react
  npx tsx logic_distribution.ts /path/to/nextjs-app --verbose
  npx tsx logic_distribution.ts /path/to/express-api --spot-check 20`);
    process.exit(0);
  }

  const projectRoot = path.resolve(args[0]);
  const verbose = args.includes("--verbose");
  const summaryOnly = args.includes("--summary-only");
  const noJson = args.includes("--no-json");

  let spotCheck = 0;
  const scIdx = args.indexOf("--spot-check");
  if (scIdx >= 0 && args[scIdx + 1]) spotCheck = parseInt(args[scIdx + 1], 10);

  let output = "analysis_results.json";
  const outIdx = args.indexOf("--output");
  if (outIdx >= 0 && args[outIdx + 1]) output = args[outIdx + 1];

  if (!fs.existsSync(projectRoot) || !fs.statSync(projectRoot).isDirectory()) {
    console.error(`Error: ${projectRoot} is not a directory`);
    process.exit(1);
  }

  console.error(`Analyzing: ${projectRoot}`);

  const results = analyzeProject(projectRoot, verbose);

  if (results.length === 0) {
    console.error("No functions found to analyze.");
    process.exit(0);
  }

  printSummary(results, projectRoot);

  if (!summaryOnly) {
    printPureFunctions(results, projectRoot);
    printBorderlineCases(results, projectRoot);
    printPerFileBreakdown(results, projectRoot);
  }

  if (spotCheck > 0) {
    printSpotCheck(results, spotCheck, projectRoot);
  }

  if (!noJson) {
    writeJsonOutput(results, output, projectRoot);
  }
}

main();

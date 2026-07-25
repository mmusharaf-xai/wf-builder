import { PrismaClient } from "@prisma/client";
import { createInMemoryPrismaClient } from "./in-memory-db";

export type StorageMode = "IN_MEMORY" | "POSTGRES";

declare global {
  // eslint-disable-next-line no-var
  var prisma: PrismaClient | undefined;
  // eslint-disable-next-line no-var
  var prismaInitPromise: Promise<PrismaClient> | undefined;
}

function getStorageMode(): StorageMode {
  const raw = (process.env.STORAGE ?? "POSTGRES").trim().toUpperCase();
  return raw === "IN_MEMORY" ? "IN_MEMORY" : "POSTGRES";
}

/** Active storage backend — driven only by STORAGE env (no query changes needed). */
export const storageMode = getStorageMode();

async function createPrismaClient(): Promise<PrismaClient> {
  if (storageMode === "IN_MEMORY") {
    console.info(
      "[db] STORAGE=IN_MEMORY — using in-process PGlite (no remote/localhost DB)"
    );
    return createInMemoryPrismaClient();
  }

  if (!process.env.PG_DB_URL) {
    throw new Error(
      "STORAGE=POSTGRES requires PG_DB_URL (e.g. postgresql://user:pass@localhost:5432/wf)"
    );
  }

  console.info("[db] STORAGE=POSTGRES — using PG_DB_URL");
  return new PrismaClient({
    datasources: {
      db: { url: process.env.PG_DB_URL },
    },
  });
}

function getOrInitClient(): Promise<PrismaClient> {
  if (globalThis.prisma) {
    return Promise.resolve(globalThis.prisma);
  }
  if (!globalThis.prismaInitPromise) {
    globalThis.prismaInitPromise = createPrismaClient().then((client) => {
      globalThis.prisma = client;
      return client;
    });
  }
  return globalThis.prismaInitPromise;
}

/**
 * Lazy PrismaClient proxy so IN_MEMORY async bootstrap works without
 * changing any call sites (`db.workflows.findMany(...)`, etc.).
 */
function createLazyPrismaClient(): PrismaClient {
  const clientPromise = getOrInitClient();

  return new Proxy({} as PrismaClient, {
    get(_target, prop) {
      if (prop === "then") {
        // Avoid treating `db` as a Promise
        return undefined;
      }
      if (typeof prop === "symbol") {
        return undefined;
      }

      const key = prop as string;

      // Client-level APIs: $connect, $disconnect, $transaction, $queryRaw, ...
      if (key.startsWith("$") || key === "constructor") {
        return async (...args: unknown[]) => {
          const client = await clientPromise;
          const value = (client as unknown as Record<string, unknown>)[key];
          if (typeof value === "function") {
            return (value as (...a: unknown[]) => unknown).apply(client, args);
          }
          return value;
        };
      }

      // Model delegates: workflows, workflowNodes, webhooks, ...
      return new Proxy(
        {},
        {
          get(_t, method) {
            if (typeof method === "symbol") return undefined;
            return async (...args: unknown[]) => {
              const client = await clientPromise;
              const model = (client as unknown as Record<string, unknown>)[key];
              if (model == null || typeof model !== "object") {
                throw new Error(`[db] Unknown Prisma model or property: ${key}`);
              }
              const fn = (model as Record<string, unknown>)[method as string];
              if (typeof fn !== "function") {
                return fn;
              }
              return (fn as (...a: unknown[]) => unknown).apply(model, args);
            };
          },
        }
      );
    },
  });
}

export const db: PrismaClient =
  globalThis.prisma ?? createLazyPrismaClient();

// Warm the singleton in dev so HMR reuses the same client / in-memory DB
if (process.env.NODE_ENV !== "production") {
  void getOrInitClient();
}

import fs from "node:fs";
import path from "node:path";
import { PGlite } from "@electric-sql/pglite";
import { PrismaPGlite } from "pglite-prisma-adapter";
import { PrismaClient, type Prisma } from "@prisma/client";

/**
 * In-process Postgres via PGlite (no remote / localhost server).
 * Same PrismaClient API — switch with STORAGE=IN_MEMORY only.
 */
export async function createInMemoryPrismaClient(): Promise<PrismaClient> {
  const pglite = new PGlite(); // ephemeral memory:// — nothing on disk or network
  const schemaPath = path.join(process.cwd(), "prisma", "in-memory-schema.sql");
  const schemaSql = fs.readFileSync(schemaPath, "utf8");
  await pglite.exec(schemaSql);

  // pglite-prisma-adapter and @prisma/client pin different @prisma/driver-adapter-utils
  // versions; runtime is compatible, so go through unknown for the options object.
  const options = {
    adapter: new PrismaPGlite(pglite),
  } as unknown as Prisma.PrismaClientOptions;

  return new PrismaClient(options);
}

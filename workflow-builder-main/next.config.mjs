/** @type {import('next').NextConfig} */
const nextConfig = {
  // Keep PGlite/WASM out of the bundler so legacy IN_MEMORY storage still loads
  // if someone runs the old Next.js API routes.
  serverExternalPackages: ["@electric-sql/pglite", "pglite-prisma-adapter"],
  experimental: {
    serverComponentsExternalPackages: [
      "@electric-sql/pglite",
      "pglite-prisma-adapter",
    ],
  },
  // Frontend talks to the Go backend via NEXT_PUBLIC_API_URL (see lib/api.ts).
  // Do not rewrite /api/* here — existing app/api/* routes would take precedence.
};

export default nextConfig;

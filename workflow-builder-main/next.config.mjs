/** @type {import('next').NextConfig} */
const nextConfig = {
  // Keep PGlite/WASM out of the bundler so IN_MEMORY storage loads correctly.
  serverExternalPackages: ["@electric-sql/pglite", "pglite-prisma-adapter"],
  experimental: {
    serverComponentsExternalPackages: [
      "@electric-sql/pglite",
      "pglite-prisma-adapter",
    ],
  },
};

export default nextConfig;

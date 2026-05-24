import type { NextConfig } from "next";

// In all environments the browser talks to /api/* on the same origin as the
// Next.js app. The dev server (and production runtime) proxies those requests
// to the Go API on the URL below. This keeps the session cookie SameSite=Lax
// and eliminates CORS/cookie hassles between web and API.
const backendURL = process.env.BACKEND_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      { source: "/api/:path*", destination: `${backendURL}/:path*` },
    ];
  },
};

export default nextConfig;

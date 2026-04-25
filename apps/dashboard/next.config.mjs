/** @type {import('next').NextConfig} */
const nextConfig = {
  eslint: {
    dirs: ["app", "components", "lib"]
  },
  experimental: {
    typedRoutes: true
  },
  images: {
    unoptimized: true
  }
};

export default nextConfig;

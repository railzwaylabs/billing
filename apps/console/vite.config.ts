import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

export default defineConfig(({ mode }) => {
  const envDir = path.resolve(__dirname, "../..");
  const env = loadEnv(mode, envDir, "");
  return {
    envDir,
    plugins: [react(), tailwindcss()],
    resolve: { alias: { "@": path.resolve(__dirname, "./src") } },
    server: {
      port: 5173,
      proxy: {
        "/admin/v1": {
          target: env.BACKEND_URL || "http://localhost:8080",
          changeOrigin: true,
        },
      },
    },
  };
});

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
	plugins: [react(), tailwindcss()],
	define: {
		__APP_VERSION__: JSON.stringify(process.env.npm_package_version || "0.0.0"),
		__GIT_COMMIT__: JSON.stringify(process.env.GIT_COMMIT || "unknown"),
		__GITHUB_URL__: JSON.stringify("https://github.com/javi11/altmount"),
	},
	build: {
		// Enable source maps for production debugging
		sourcemap: false,
		// Minify with esbuild for faster builds
		minify: "esbuild",
		// Target modern browsers for smaller bundles
		target: "es2020",
		// Rollup options for manual chunking
		rollupOptions: {
			output: {
				// Manual chunk splitting for better caching
				manualChunks: {
					// Vendor chunk for React core
					"vendor-react": ["react", "react-dom", "react-router-dom"],
					// React Query in separate chunk (used across the app)
					"vendor-query": ["@tanstack/react-query"],
					// Charts in separate chunk (heavy, lazy loaded)
					"vendor-charts": ["recharts"],
					// Form handling
					"vendor-forms": ["react-hook-form", "@hookform/resolvers", "zod"],
					// Icons (tree-shakeable but used frequently)
					"vendor-icons": ["lucide-react"],
				},
			},
		},
		// Increase chunk size warning limit slightly
		chunkSizeWarningLimit: 600,
	},
	server: {
		port: 5173,
		strictPort: true,
		proxy: {
			"/api": {
				target: "http://localhost:8080",
				changeOrigin: true,
				ws: true,
			},
			"/sabnzbd": {
				target: "http://localhost:8080",
				changeOrigin: true,
			},
			"/webdav": {
				target: "http://localhost:8080",
				changeOrigin: true,
			},
		},
	},
	// Optimize dependency pre-bundling
	optimizeDeps: {
		include: ["react", "react-dom", "react-router-dom", "@tanstack/react-query", "lucide-react"],
	},
});

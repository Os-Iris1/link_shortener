import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const proxy = {
  '/api': 'http://127.0.0.1:8080',
  '/healthz': 'http://127.0.0.1:8080',
  '^/[A-Za-z0-9_-]{12}$': 'http://127.0.0.1:8080',
};

export default defineConfig({
  plugins: [react()],
  server: { port: 5173, strictPort: true, proxy },
  preview: { proxy },
});

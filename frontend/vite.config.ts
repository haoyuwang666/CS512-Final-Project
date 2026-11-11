// TODO(Tian): Add controller/Peer proxy routes when their dashboard panels are implemented.
import { defineConfig } from 'vite';
export default defineConfig({ server: { proxy: { '/api': 'http://127.0.0.1:8080' } }, build: { outDir: 'dist' } });

import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
export default defineConfig({plugins:[react()],server:{port:5173,proxy:{'/api':process.env.VITE_API_PROXY??'http://localhost:8080'}},build:{sourcemap:true},test:{environment:'jsdom',globals:true,exclude:['e2e/**','node_modules/**'],setupFiles:'./src/test/setup.ts',css:true,coverage:{provider:'v8',reporter:['text','html']}}});

import {defineConfig} from '@playwright/test';
export default defineConfig({testDir:'./e2e',timeout:30_000,retries:1,use:{baseURL:'http://127.0.0.1:4173',trace:'retain-on-failure'},webServer:{command:'npm run build && npx vite preview --host 127.0.0.1',url:'http://127.0.0.1:4173',reuseExistingServer:false,timeout:120_000}});

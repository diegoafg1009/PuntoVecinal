import { defineConfig } from '@hey-api/openapi-ts'

// Generates the typed API client from the backend contract. Run
// `make bundle` in `backend/` first so the bundled file exists.
export default defineConfig({
  input: '../openapi/dist/openapi.yaml',
  output: 'app/api',
  plugins: [
    '@hey-api/typescript',
    '@hey-api/client-fetch',
    '@hey-api/sdk',
    '@tanstack/vue-query'
  ]
})

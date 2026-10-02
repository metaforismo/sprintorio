# Sprintorio application

Svelte 5 + SvelteKit, TypeScript, Tailwind, and Paraglide translations. The application connects to the Go API.

```sh
npm ci
npm run dev
```

Development uses port 5173 and proxies `/api` and `/uploads` to localhost:8080. Configure the root environment before starting the backend.

```sh
npm run check
npm run test:unit
npm run test:e2e
npm run build
```

Playwright builds a static preview and mocks API responses; real persistence is tested by the backend integration suite. `PLAYWRIGHT_CHANNEL=chrome` selects an installed Chrome browser. See [../TESTING.md](../TESTING.md).

The production image serves the static app with Caddy on port 3000. Set `API_URL` to the reachable Go backend origin. The repository Compose stacks configure this automatically.

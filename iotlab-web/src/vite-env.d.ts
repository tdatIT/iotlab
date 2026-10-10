/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** iotlab-service API base, e.g. https://host/api/v1. Defaults to http://localhost:3000/api/v1. */
  readonly VITE_API_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

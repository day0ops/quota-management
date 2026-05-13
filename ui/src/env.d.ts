/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ENABLE_RATE_LIMITS?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare const __APP_VERSION__: string;
declare const __BUILD_TIME__: string;

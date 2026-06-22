import type { $Fetch } from "ofetch";

let authToken: string | null = null;
let apiKey: string | null = null;

export function setAuthHeader(token: string | null) {
  authToken = token;
}

export function setApiKey(key: string | null) {
  apiKey = key;
}

function createApiFetch(): $Fetch {
  const config = useRuntimeConfig();
  const publicConfig = config.public as Record<string, string>;
  const baseURL = publicConfig.apiBaseUrl;

  return $fetch.create({
    baseURL,
    headers: {
      "X-Client-Version": publicConfig.clientVersion || "dev",
    },
    onRequest({ options }) {
      const headers = (options.headers ||= {}) as Record<string, string>;
      if (authToken) {
        headers.Authorization = `Bearer ${authToken}`;
      }
      if (apiKey) {
        headers["x-api-key"] = apiKey;
      }
    },
  });
}

export function useApi() {
  return { apiFetch: createApiFetch(), setAuthHeader, setApiKey };
}

export function useApiComposable() {
  return {
    useApi: createApiFetch,
  };
}

let authToken: string | null = null;
let apiKey: string | null = null;

export function setAuthHeader(token: string | null) {
  authToken = token;
}

export function setApiKey(key: string | null) {
  apiKey = key;
}

function createApiFetch() {
  const config = useRuntimeConfig();
  const baseURL = config.public.apiBaseUrl as string;

  return $fetch.create({
    baseURL,
    headers: {
      "X-Client-Version": "web",
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

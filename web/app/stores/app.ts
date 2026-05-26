import { defineStore } from "pinia";

export interface AppData {
  url: string;
  name: string;
  env: string;
  appDownloadUrl: string;
  documentationUrl: string;
  githubUrl: string;
}

export const useAppStore = defineStore("app", () => {
  const config = useRuntimeConfig();
  const polling = ref(false);

  const appData = computed<AppData>(() => {
    let url = (config.public.appUrl as string) || "";
    if (url.length > 0 && url[url.length - 1] === "/") {
      url = url.substring(0, url.length - 1);
    }
    return {
      url,
      env: config.public.appEnv as string,
      appDownloadUrl: config.public.appDownloadUrl as string,
      documentationUrl: config.public.appDocumentationUrl as string,
      githubUrl: config.public.appGithubUrl as string,
      name: config.public.appName as string,
    };
  });

  const isLocal = computed(() => config.public.appEnv === "local");

  function setPolling(value: boolean) {
    polling.value = value;
  }

  return {
    polling,
    appData,
    isLocal,
    setPolling,
  };
});

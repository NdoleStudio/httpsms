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
    const publicConfig = config.public as Record<string, string>;
    let url = publicConfig.appUrl || "";
    if (url.endsWith("/")) {
      url = url.substring(0, url.length - 1);
    }
    return {
      url,
      env: publicConfig.appEnv,
      appDownloadUrl: publicConfig.appDownloadUrl,
      documentationUrl: publicConfig.appDocumentationUrl,
      githubUrl: publicConfig.appGithubUrl,
      name: publicConfig.appName,
    };
  });

  const isLocal = computed(
    () => (config.public as Record<string, string>).appEnv === "local",
  );

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

import { defineStore } from "pinia";
import { setAuthHeader, setApiKey } from "~/composables/useApi";
import type { User } from "~~/shared/types/user";

export interface AuthUser {
  email: string | null;
  displayName: string | null;
  id: string;
}

export const useAuthStore = defineStore("auth", () => {
  const authStateChanged = ref(false);
  const authUser = ref<AuthUser | null>(null);
  const user = ref<User | null>(null);
  const { apiFetch } = useApi();

  async function setAuthUserAction(newUser: AuthUser | null | undefined) {
    const userChanged = newUser?.id !== authUser.value?.id;
    authUser.value = newUser ?? null;
    authStateChanged.value = true;

    if (userChanged && newUser !== null) {
      await Promise.all([loadUser(), loadPhones()]);
    }
  }

  async function onAuthStateChanged(firebaseUser: any) {
    if (firebaseUser == null) {
      authUser.value = null;
      user.value = null;
      authStateChanged.value = true;
      setApiKey("");
      return;
    }
    setAuthHeader(await firebaseUser.getIdToken());
    const { uid, email, displayName } = firebaseUser;
    authUser.value = { id: uid, email, displayName };
    authStateChanged.value = true;
  }

  async function onIdTokenChanged(firebaseUser: any) {
    if (firebaseUser == null) {
      setApiKey("");
      return;
    }
    setAuthHeader(await firebaseUser.getIdToken());
  }

  async function loadUser() {
    const response = await apiFetch<{ data: User }>("/v1/users/me");
    user.value = response.data;
  }

  async function updateUser(payload: { owner?: string; timezone?: string }) {
    const phonesStore = usePhonesStore();
    if (payload.owner) {
      phonesStore.setOwner(payload.owner);
    }

    const activePhone = phonesStore.activePhone;
    if (!activePhone) return;

    const response = await apiFetch<{ data: User }>("/v1/users/me", {
      method: "PUT",
      body: {
        active_phone_id: activePhone.id,
        timezone: payload.timezone ?? user.value?.timezone,
      },
    });

    setApiKey(response.data.api_key);
    user.value = response.data;
  }

  async function deleteUserAccount(): Promise<string> {
    const response = await apiFetch<{ message: string }>("/v1/users/me", {
      method: "DELETE",
    });
    return response.message;
  }

  async function rotateApiKey(userId: string): Promise<User> {
    const response = await apiFetch<{ data: User }>(
      `/v1/users/${userId}/api-keys`,
      {
        method: "DELETE",
      },
    );
    user.value = response.data;
    setApiKey(response.data.api_key);
    return response.data;
  }

  function resetState() {
    user.value = null;
    authUser.value = null;
    authStateChanged.value = true;
    setApiKey("");
  }

  function loadPhones() {
    const phonesStore = usePhonesStore();
    return phonesStore.loadPhones(false);
  }

  return {
    authStateChanged,
    authUser,
    user,
    setAuthUserAction,
    onAuthStateChanged,
    onIdTokenChanged,
    loadUser,
    updateUser,
    deleteUserAccount,
    rotateApiKey,
    resetState,
  };
});

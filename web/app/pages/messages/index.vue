<script setup lang="ts">
import { mdiArrowLeft, mdiSend, mdiCircle } from "@mdi/js";

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "New Message - httpSMS",
});

const router = useRouter();
const { mdAndDown } = useDisplay();
const notificationsStore = useNotificationsStore();
const phonesStore = usePhonesStore();
const messagesStore = useMessagesStore();
const { useApi } = useApiComposable();

const sending = ref(false);
const formPhoneNumber = ref("");
const formContent = ref("");
const formAttachments = ref("");
const errors = ref(new Map<string, string[]>());

async function sendMessage() {
  errors.value = new Map();
  sending.value = true;

  try {
    const api = useApi();
    await api("/v1/messages/send", {
      method: "POST",
      body: {
        to: formPhoneNumber.value,
        from: phonesStore.owner,
        content: formContent.value,
        attachments: formAttachments.value
          .trim()
          .split(",")
          .filter((x) => x.trim() !== "")
          .map((x) => x.trim()),
      },
    });
    notificationsStore.addNotification({
      message: "Message Sent!",
      type: "success",
    });
    await router.push("/threads");
  } catch (err: any) {
    const data = err?.data?.data;
    if (data) {
      const newErrors = new Map<string, string[]>();
      if (data.content) newErrors.set("content", data.content);
      if (data.to)
        newErrors.set(
          "to",
          data.to.map((x: string) =>
            x.replace("to field", "phone number field"),
          ),
        );
      if (data.attachments) newErrors.set("attachments", data.attachments);
      if (data.from) {
        notificationsStore.addNotification({
          message: data.from[0],
          type: "error",
        });
      }
      errors.value = newErrors;
    }
  } finally {
    sending.value = false;
  }
}
</script>

<template>
  <VContainer fluid class="pa-0" :class="{ 'fill-height': true }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>
          New Message
          <VIcon
            size="x-small"
            class="mx-2"
            color="primary"
            :icon="mdiCircle"
          />
          {{ useFilters().phoneNumber(phonesStore.owner) }}
        </VToolbarTitle>
      </VAppBar>
      <VContainer class="mt-16">
        <VRow>
          <VCol cols="12" md="8" offset-md="2" xl="6" offset-xl="3">
            <form @submit.prevent="sendMessage">
              <VTextField
                v-model="formPhoneNumber"
                :disabled="sending"
                :error="errors.has('to')"
                :error-messages="errors.get('to')"
                variant="outlined"
                persistent-placeholder
                placeholder="Recipient phone number e.g +18005550199"
                label="Phone Number"
              />
              <VTextarea
                v-model="formContent"
                :error="errors.has('content')"
                :error-messages="errors.get('content')"
                :disabled="sending"
                variant="outlined"
                persistent-placeholder
                placeholder="Enter your message here"
                label="Content"
              />
              <VTextarea
                v-model="formAttachments"
                :error="errors.has('attachments')"
                :error-messages="errors.get('attachments')"
                :disabled="sending"
                variant="outlined"
                rows="2"
                class="mb-8"
                persistent-placeholder
                persistent-hint
                hint="The message will be sent as an MMS when a comma separated list of attachment URLs are present"
                placeholder="https://example.com/image.jpg, https://example.com/video.mp4"
                label="Attachment URLs (optional)"
              />
              <VBtn
                type="submit"
                color="primary"
                :disabled="sending"
                :block="mdAndDown"
              >
                <VIcon :icon="mdiSend" />
                Send Message
              </VBtn>
            </form>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>

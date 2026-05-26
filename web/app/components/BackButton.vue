<script setup lang="ts">
import { useDisplay } from "vuetify";
import { mdiArrowLeft } from "@mdi/js";
import type { RouteLocationRaw } from "vue-router";

const props = withDefaults(
  defineProps<{
    route?: RouteLocationRaw;
    block?: boolean;
  }>(),
  {
    block: false,
  },
);

const router = useRouter();
const { smAndDown } = useDisplay();

function goBack() {
  if (props.route) {
    router.push(props.route);
    return;
  }
  if (window.history.length > 1) {
    router.back();
    return;
  }
  router.push({ name: "index" });
}
</script>

<template>
  <v-btn
    color="default"
    :size="smAndDown ? 'small' : 'default'"
    :block="block"
    @click="goBack"
  >
    <v-icon :icon="mdiArrowLeft" />
    Go Back
  </v-btn>
</template>

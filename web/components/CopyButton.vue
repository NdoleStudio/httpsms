<template>
  <v-btn
    :disabled="disabled"
    :color="color"
    :small="$vuetify.breakpoint.smAndDown"
    :block="block"
    :large="large"
    @click="copy"
  >
    <v-icon>mdi-content-copy</v-icon>
    {{ copyText }}
  </v-btn>
</template>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { NotificationRequest } from '~/store'
@Component
export default class CopyButton extends Vue {
  @Prop({ required: true, type: String }) value!: string
  @Prop({ required: false, type: String, default: 'default' }) color!: string
  @Prop({ required: false, type: Boolean, default: false }) block!: boolean
  @Prop({ required: false, type: Boolean, default: false }) large!: boolean
  @Prop({ required: false, type: String, default: 'Copy' }) copyText!: string
  @Prop({ required: false, type: String, default: 'Copied' })
  notificationText!: string

  disabled = false

  async copy() {
    this.disabled = true
    await navigator.clipboard.writeText(this.value)

    await this.$store.dispatch('addNotification', {
      message: this.notificationText,
      type: 'success',
    } as NotificationRequest)

    setTimeout(() => {
      this.disabled = false
    }, 5000)
  }
}
</script>

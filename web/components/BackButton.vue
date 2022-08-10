<template>
  <v-btn
    color="default"
    :small="$vuetify.breakpoint.smAndDown"
    :block="block"
    @click="goBack"
  >
    <v-icon>{{ mdiArrowLeft }}</v-icon>
    Go Back
  </v-btn>
</template>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { Location } from 'vue-router'
import { mdiArrowLeft } from '@mdi/js'
@Component
export default class BackButton extends Vue {
  @Prop({ required: false }) route?: Location
  @Prop({ required: false, type: Boolean, default: false }) block!: boolean
  mdiArrowLeft = mdiArrowLeft
  goBack(): void {
    if (this.route) {
      this.$router.push(this.route)
      return
    }
    if (window.history.length > 1) {
      this.$router.back()
      return
    }
    this.$router.push({ name: 'index' })
  }
}
</script>

<template>
  <v-btn
    :block="block"
    :type="type"
    :small="small"
    :large="large"
    :x-large="xLarge"
    :color="color"
    :text="text"
    :tile="tile"
    exact
    :disabled="isLoading"
    @click.prevent="onClick"
  >
    <v-progress-circular
      v-if="isClicked"
      :size="small ? 20 : 25"
      color="grey"
      class="mr-2"
      indeterminate
    ></v-progress-circular>
    <v-icon v-if="icon && !isLoading" left>{{ icon }}</v-icon>
    <slot></slot>
  </v-btn>
</template>

<script lang="ts">
import { Component, PropSync, Prop, Vue, Watch } from 'vue-property-decorator'
@Component
export default class SocialButtons extends Vue {
  @Prop({ required: false, type: String, default: 'submit' }) type!: string
  @Prop({ required: false, type: Boolean, default: false }) block!: boolean
  @Prop({ required: false, type: Boolean, default: false }) large!: boolean
  @Prop({ required: false, type: Boolean, default: false }) xLarge!: boolean
  @Prop({ required: false, type: Boolean, default: false }) tile!: boolean
  @Prop({ required: false, type: Boolean, default: false }) text!: boolean
  @Prop({ required: false, type: Boolean, default: false }) small!: boolean
  @Prop({ required: false, type: String, default: 'primary' }) color!: string
  @Prop({ required: false, type: String, default: null }) icon!: string | null
  @PropSync('loading', { required: true, type: Boolean }) isLoading!: boolean

  isClicked = false

  @Watch('loading')
  onChildChanged(submitting: boolean) {
    if (!submitting && this.isClicked) {
      this.isClicked = false
    }
  }

  onClick() {
    this.isClicked = true
    this.$emit('click')
  }
}
</script>

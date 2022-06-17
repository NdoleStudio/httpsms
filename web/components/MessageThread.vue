<template>
  <v-list two-line class="px-0 py-0" subheader>
    <v-list-item-group>
      <template v-for="thread in threads">
        <v-list-item
          :key="thread.id"
          :to="'/threads/' + thread.id"
          class="py-1"
        >
          <v-list-item-avatar :color="thread.color">
            <v-icon dark> mdi-account </v-icon>
          </v-list-item-avatar>
          <v-list-item-content>
            <v-list-item-title>
              {{ thread.contact }}
            </v-list-item-title>
            <v-list-item-subtitle>
              {{ thread.last_message_content }}
            </v-list-item-subtitle>
          </v-list-item-content>
          <v-list-item-action>
            <v-list-item-action-text>
              {{ threadDate(thread.order_timestamp) }}
            </v-list-item-action-text>
          </v-list-item-action>
        </v-list-item>
      </template>
    </v-list-item-group>
  </v-list>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

@Component
export default class MessageThread extends Vue {
  get threads(): Array<MessageThread> {
    return this.$store.getters.getThreads
  }

  threadDate(date: string): string {
    return new Date(date).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
    })
  }
}
</script>

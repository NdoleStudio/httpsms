<template>
  <v-list two-line class="px-0 py-0" subheader>
    <v-list-item-group v-model="selected">
      <template v-for="thread in threads">
        <v-list-item :key="thread.id" class="py-1">
          <v-list-item-avatar :color="thread.color">
            <v-icon dark> mdi-account </v-icon>
          </v-list-item-avatar>
          <v-list-item-content>
            <v-list-item-title>
              {{ thread.phoneNumber }}
            </v-list-item-title>
            <v-list-item-subtitle>
              {{ thread.message }}
            </v-list-item-subtitle>
          </v-list-item-content>
          <v-list-item-action>
            <v-list-item-action-text>
              {{ thread.date }}
            </v-list-item-action-text>
          </v-list-item-action>
        </v-list-item>
      </template>
    </v-list-item-group>
  </v-list>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

type ThreadItem = {
  id: string
  phoneNumber: string
  message: string
  date: string
  color: string
}

@Component
export default class MessageThread extends Vue {
  selected: string = '1'
  get threads(): Array<ThreadItem> {
    return [
      {
        id: '1',
        phoneNumber: '+1 800 5550 100',
        message:
          'I was calling about the apartment? Is it still available and can I see it today?',
        date: 'Fri',
        color: 'indigo',
      },
      {
        id: '2',
        phoneNumber: '+1 800 5550 500',
        message:
          'I was calling about the apartment? Is it still available and can I see it today?',
        date: '20 May',
        color: 'teal',
      },
    ]
  }
}
</script>

<template>
  <v-container>
    <v-row class="mt-16">
      <v-col cols="12" md="9">
        <v-hover v-for="post in blogPosts" v-slot="{ hover }" :key="post.route">
          <vue-glow
            color="#329ef4"
            mode="hex"
            elevation="7"
            :intensity="hover ? 0.7 : 0"
            intense
          >
            <v-card :to="post.route">
              <v-card-title class="text-h4 text-break">{{
                post.title
              }}</v-card-title>
              <v-card-subtitle>
                <span class="text-uppercase blue--text">{{ post.date }}</span>
                • <span class="text-uppercase">{{ post.readTime }}</span>
              </v-card-subtitle>
              <v-card-text class="mt-n2">
                <p class="text--primary subtitle-1">{{ post.description }}</p>
                <div class="d-flex mt-n2">
                  <v-avatar class="mb-n2">
                    <v-img :src="post.authorImage"></v-img>
                  </v-avatar>
                  <div class="ml-2">
                    <p class="subtitle-1">{{ post.authorName }}</p>
                    <p class="mt-n5 mb-n4">
                      {{ post.authorTwitter }}
                      <v-icon color="#1DA1F2" small>{{ mdiTwitter }}</v-icon>
                    </p>
                  </div>
                </div>
              </v-card-text>
            </v-card>
          </vue-glow>
        </v-hover>
      </v-col>
      <v-col v-if="$vuetify.breakpoint.mdAndUp" md="3">
        <blog-info></blog-info>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { mdiTwitter } from '@mdi/js'

export default {
  name: 'BlogIndex',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      blogPosts: [
        {
          route: '/blog/forward-incoming-sms-from-phone-to-webhook',
          title:
            'How to forward a text message (SMS) from an android phone into your webhook',
          date: 'April 08, 2023',
          readTime: '5 min read',
          authorImage: require('@/assets/img/arnold.png'),
          description:
            'You can now program your android phone to forward messages received on your phone to your server and trigger powerful automations with tools like zapper and IFTTFT.',
          authorName: 'Acho Arnold',
          authorTwitter: 'acho_arnold',
        },
      ],
    }
  },
  head() {
    return {
      title: 'Blog Posts - SuperButton',
    }
  },
}
</script>

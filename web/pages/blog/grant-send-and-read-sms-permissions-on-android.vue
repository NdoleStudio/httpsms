<template>
  <v-container class="pt-8">
    <v-row class="mt-16">
      <v-col cols="12" md="9">
        <h1
          class="mt-1"
          :class="{
            'text-h2': $vuetify.breakpoint.mdAndUp,
            'text-h3': !$vuetify.breakpoint.mdAndUp,
          }"
        >
          How to grant SEND_SMS and RECEIVE_SMS permissions on Android 15+
        </h1>
        <p class="subtitle-2 mt-2">
          <span class="text-uppercase blue--text">{{ postDate }}</span>
          • <span class="text-uppercase">{{ readTime }}</span>
        </p>
        <p class="text--secondary subtitle-1 mt-2">
          In Android 15 (Vanilla Ice Cream), the
          <code>android.permission.SEND_SMS</code> and
          <code>android.permission.SEND_SMS</code> permissions are now hard
          restricted and cannot be granted
          <a
            class="text-decoration-none"
            href="https://developer.android.com/training/permissions/requesting"
            >via the runtime permissions interface</a
          >.
        </p>

        <v-alert type="warning" outlined>
          Granting the <code>SEND_SMS</code> and
          <code>RECEIVE_SMS</code> permissions will allow an Android app to be
          able to read and send SMS messages on your phone. Make sure you trust
          the application before allowing these permissions.
        </v-alert>
        <h3 class="text-h4 mt-8 mb-2">Step1: Open App Info</h3>
        <p>
          Long press the icon of the android app which you want to grant the
          permission and select <b>"App info"</b>
        </p>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/grant-send-and-read-sms-permissions-on-android/app-info.png')
          "
        ></v-img>
        <h3 class="text-h4 mb-4 mt-16">Step 2: Allow Restricted Permissions</h3>
        <p>
          On the App Info page, click on the menu button
          <v-icon>{{ mdiDotsVertical }}</v-icon> and select the
          <b>"Allow restricted settings"</b> option
        </p>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/grant-send-and-read-sms-permissions-on-android/allow-restricted-settings.png')
          "
        ></v-img>
        <h3 class="text-h4 mb-4 mt-16">Step 3: Allow SMS Permissions</h3>
        <p>
          Once you have allowed the restricted settings from step 2 above, You
          can navigate to <b>Permissions ➡️ SMS</b> and tap the Allow button to
          grant SMS permissions to the android app.
        </p>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/grant-send-and-read-sms-permissions-on-android/allow.png')
          "
        ></v-img>
        <h3 class="text-h4 mt-12">Conclusion</h3>
        <p>
          Congratulations, you have successfully configured SMS permissions on
          your Android app. Don't hesitate to contact us if you face any
          problems while following this guide.
        </p>
        <blog-author-bio></blog-author-bio>
        <v-divider class="mx-16"></v-divider>
        <div class="text-center mt-8 mb-4">
          <back-button></back-button>
        </div>
      </v-col>
      <v-col v-if="$vuetify.breakpoint.mdAndUp" md="3">
        <blog-info></blog-info>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import {
  mdiLanguageGo,
  mdiTwitter,
  mdiDotsVertical,
  mdiLanguageJavascript,
} from '@mdi/js'
export default {
  name: 'GrantSendAndReadSmsPermissionsOnAndroid',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      mdiLanguageGo,
      mdiLanguageJavascript,
      mdiDotsVertical,
      selectedTab: 'javascript',
      authorImage: require('@/assets/img/arnold.png'),
      authorName: 'Acho Arnold',
      postDate: 'February 18, 2025',
      readTime: '5 min read',
      authorTwitter: 'acho_arnold',
    }
  },
  head() {
    return {
      title:
        'How to grant SEND_SMS and RECEIVE_SMS permissions on Android 14+ - httpSMS',
      meta: [
        {
          hid: 'og:title',
          property: 'og:title',
          content:
            'How to grant SEND_SMS and RECEIVE_SMS permissions on Android 14+',
        },
        {
          hid: 'og:description',
          property: 'og:description',
          content:
            'We have added support for end-to-end encryption for SMS messages so that no one can see the content of the messages you send using httpSMS except you.',
        },
      ],
    }
  },
}
</script>

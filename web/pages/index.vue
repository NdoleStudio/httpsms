<template>
  <div>
    <v-app-bar elevation="0" color="#121212" height="70" fixed>
      <v-container>
        <v-row>
          <v-col class="w-full d-flex">
            <h2
              class="text--primary text-h4"
              :class="{
                'mt-5': $vuetify.breakpoint.mdAndDown,
              }"
            >
              <v-avatar
                v-if="$vuetify.breakpoint.lgAndUp"
                size="70"
                class="mt-4"
              >
                <v-img
                  class="mb-4"
                  contain
                  :src="require('assets/img/icon.svg')"
                ></v-img>
              </v-avatar>
              Http SMS
            </h2>
            <v-spacer></v-spacer>
            <v-btn
              exact-path
              class="primary mt-5 mb-5"
              :large="$vuetify.breakpoint.lgAndUp"
              :to="{ name: 'login' }"
            >
              Get Started
              <span v-if="$vuetify.breakpoint.lgAndUp">&nbsp;For Free</span>
            </v-btn>
          </v-col>
        </v-row>
      </v-container>
    </v-app-bar>
    <v-container class="pb-16">
      <v-row :class="{ 'py-16': $vuetify.breakpoint.lgAndUp }">
        <v-col
          cols="12"
          md="4"
          :class="{
            'text-center pt-16': $vuetify.breakpoint.mdAndDown,
            'py-16': $vuetify.breakpoint.lgAndUp,
          }"
        >
          <h1
            class="text-h2 font-weight-bold gradient-header"
            :class="{
              'mt-16': $vuetify.breakpoint.lgAndUp,
              'mt-10': $vuetify.breakpoint.mdAndUp,
            }"
          >
            Convert your android phone into an SMS gateway.
          </h1>
          <h2 class="text--secondary text-h5 mt-8 mb-8">
            Use your android phone to send and receive SMS messages using an
            intuitive HTTP API.
          </h2>
          <div :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }">
            <v-btn
              color="primary"
              large
              exact-path
              class="mt-4 mb-4"
              :to="{ name: 'login' }"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp">{{ mdiSend }}</v-icon>
              Get Started
            </v-btn>
            <v-btn
              large
              color="default"
              target="_blank"
              class="mt-4 mb-4 ml-4"
              :href="$store.getters.getAppData.githubUrl"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp">
                {{ mdiGithub }}
              </v-icon>
              Open Source
            </v-btn>
          </div>
          <div
            class="mt-4"
            :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }"
          >
            <v-icon color="success">
              {{ mdiCheckCircle }}
            </v-icon>
            Free to use
            <v-icon class="ml-4" color="success">
              {{ mdiCheckCircle }}
            </v-icon>
            100% Open Source
          </div>
          <v-divider
            v-if="$vuetify.breakpoint.mdAndDown"
            class="mt-6 mr-16 success"
            :class="{ 'ml-16': $vuetify.breakpoint.mdAndDown }"
          ></v-divider>
        </v-col>
        <v-col cols="12" md="8" class="py-16">
          <v-img
            class="mx-auto mb-4"
            max-height="700"
            max-width="98%"
            contain
            :src="require('assets/img/writing-code-phone.svg')"
          ></v-img>
        </v-col>
      </v-row>
    </v-container>
    <v-sheet class="py-16">
      <v-container>
        <v-row class="mb-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="2">
            <div>
              <h3 class="text-h3 mb-1">Control Sending</h3>
              <h5 class="text-h6 font-weight-light">
                Send bulk messages without mobile carrier limitations. If you
                set a rate e.g 3 messages per minute, we will queue up your
                messages and send them at a rate of 1 message per 20 seconds.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/queue.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mb-16">
          <v-col cols="12" md="6" class="d-flex align-center">
            <div>
              <h3 class="text-h3 mb-1">Monitoring</h3>
              <h5 class="text-h6 font-weight-light">
                If your android phone goes offline for some reason and it can't
                send SMS messages, we will send you a notification immediately.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/alert.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mt-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="2">
            <div>
              <h3 class="text-h3 mb-1">Open Source</h3>
              <h5 class="text-h6 font-weight-light">
                HttpSms is transparent and fully open source. The source code is
                available
                <a
                  target="_blank"
                  class="text-decoration-none"
                  :href="$store.getters.getAppData.githubUrl"
                  >on GitHub under the MIT License</a
                >. Feel free to fork it, verify it or submit a pull request.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/open-source.svg')"
            ></v-img>
          </v-col>
        </v-row>
      </v-container>
    </v-sheet>
    <v-container class="pb-16">
      <v-row class="mt-16">
        <v-col>
          <h2 class="text-h3 text-center">Get Started</h2>
        </v-col>
      </v-row>
      <v-row class="pb-16">
        <v-col cols="12">
          <v-row>
            <v-col cols="12" md="5" class="pr-4">
              <v-timeline dense class="mt-10 ml-n4">
                <v-timeline-item color="primary" :icon="mdiTallyMark1">
                  <v-card
                    class="elevation-2"
                    @mouseover="step1Hover = true"
                    @mouseleave="step1Hover = false"
                  >
                    <v-card-title class="text-h5"> Step 1 </v-card-title>
                    <v-card-text class="subtitle-1">
                      <router-link
                        class="font-weight-bold text-decoration-none"
                        :to="{ name: 'login' }"
                      >
                        Create an account
                      </router-link>
                      on httpsms.com and obtain you API key on the settings
                      page.
                    </v-card-text>
                  </v-card>
                </v-timeline-item>
                <v-timeline-item color="primary" :icon="mdiTallyMark2">
                  <v-card
                    class="elevation-2"
                    @mouseover="step2Hover = true"
                    @mouseleave="step2Hover = false"
                  >
                    <v-card-title class="text-h5"> Step 2 </v-card-title>
                    <v-card-text class="subtitle-1">
                      <a
                        download
                        class="font-weight-bold text-decoration-none"
                        :href="$store.getters.getAppData.appDownloadUrl"
                        >Download</a
                      >
                      and install the companion android application on your
                      phone and sign in using your API Key.
                    </v-card-text>
                  </v-card>
                </v-timeline-item>
                <v-timeline-item color="primary" :icon="mdiTallyMark3">
                  <v-card class="elevation-2">
                    <v-card-title class="text-h5"> Step 3 </v-card-title>
                    <v-card-text class="subtitle-1">
                      Start sending and receiving SMS messages using our rich
                      HTTP API. You can find the documentation on
                      <a
                        class="text-decoration-none"
                        :href="$store.getters.getAppData.documentationUrl"
                      >
                        {{ $store.getters.getAppData.documentationUrl }}
                      </a>
                    </v-card-text>
                  </v-card>
                </v-timeline-item>
              </v-timeline>
            </v-col>
            <v-col
              cols="12"
              md="7"
              :class="{ 'pt-16 pb-16': $vuetify.breakpoint.lgAndUp }"
            >
              <div class="w-full mt-4">
                <v-tabs v-model="selectedTab" show-arrows>
                  <v-tab href="#javascript">
                    <v-icon color="#efd81d" class="mr-1">{{
                      mdiLanguageJavascript
                    }}</v-icon>
                    Javascript
                  </v-tab>
                  <v-tab href="#php">
                    <v-icon color="#777bb3" class="mr-2">{{
                      mdiLanguagePhp
                    }}</v-icon>
                    PHP
                  </v-tab>
                  <v-tab href="#go">
                    <v-icon color="#00aed8" class="mr-2">{{
                      mdiLanguageGo
                    }}</v-icon>
                    Go
                  </v-tab>
                  <v-tab href="#java">
                    <v-icon color="#0c89c7" class="mr-2">{{
                      mdiLanguageJava
                    }}</v-icon>
                    Java
                  </v-tab>
                  <v-tab href="#curl">
                    <v-icon color="primary" class="mr-2">{{
                      mdiPowershell
                    }}</v-icon>
                    cURL
                  </v-tab>
                  <v-tab href="#c-sharp">
                    <v-icon color="#68217a" class="mr-2">{{
                      mdiLanguageCsharp
                    }}</v-icon>
                    c-sharp
                  </v-tab>
                </v-tabs>
                <v-tabs-items v-model="selectedTab">
                  <v-tab-item value="javascript">
                    <pre v-highlight class="javascript w-full mt-n2 mb-n13">
<code>let apiKey = "Get API Key from https://httpsms.com/settings";

fetch('https://api.httpsms.com/v1/messages/send', {
    method: 'POST',
    headers: {
        'x-api-key': apiKey,
        'Accept': 'application/json',
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({
        "content": "This is a sample text message",
        "from": "+18005550199",
        "to": "+18005550100"
    })
})
.then(res => res.json())
.then((data) => console.log(data));
</code>
                    </pre>
                  </v-tab-item>
                  <v-tab-item value="php">
                    <pre v-highlight class="php w-full mt-n2 mb-n13">
<code>&#60;?php
// initialize guzzle client https://github.com/guzzle/guzzle
$client = new GuzzleHttp\Client();

$apiKey = "Get API Key from https://httpsms.com/settings";

$res = $client->request('POST', 'https://api.httpsms.com/v1/messages/send', [
  'headers' => [
    'x-api-key' => $apiKey,
  ],
  'json'    => [
    'content' => 'This is a sample text message',
    'from'    => "+18005550199",
    'to'      => '+18005550100'
  ]
]);

echo $res->getBody();
</code>
                    </pre>
                  </v-tab-item>
                  <v-tab-item value="go">
                    <pre v-highlight class="go w-full mt-n2 mb-n13">
<code>import "github.com/NdoleStudio/httpsms-go"

client := htpsms.New(htpsms.WithAPIKey(/* API Key from https://httpsms.com/settings */))

client.Messages.Send(context.Background(), &httpsms.MessageSendParams{
    Content: "This is a sample text message",
    From:    "+18005550199",
    To:      "+18005550100",
})
</code>
                    </pre>
                  </v-tab-item>
                  <v-tab-item value="java">
                    <pre v-highlight class="java w-full mt-n2 mb-n13">
<code>var client = HttpClient.newHttpClient();

var apiKey = "Get API Key from https://httpsms.com/settings";

JSONObject request = new JSONObject();
request.put("content", "This is a sample text message");
request.put("from", "+18005550199")
request.put("to", "+18005550100")

// create a request
var request = HttpRequest.newBuilder()
  .uri(URI.create("https://api.httpsms.com/v1/messages/send"))
  .header("accept", "application/json")
  .header("x-api-key", apiKey)
  .setEntity(new StringEntity(request.toString()))
  .POST()
  .build();

// use the client to send the request
var response = client.send(request, new JsonBodyHandler&#60;&#62;(APOD.class));

// the response:
System.out.println(response.body().get());
</code>
                    </pre>
                  </v-tab-item>
                  <v-tab-item value="curl">
                    <pre v-highlight class="bash w-full mt-n2 mb-n13">
<code>curl --location --request POST 'https://api.httpsms.com/v1/messages/send' \
--header 'x-api-key: Get API Key from https://httpsms.com/settings' \
--header 'Content-Type: application/json' \
--data-raw '{
    "from": "+18005550199",
    "to": "+18005550100",
    "content": "This is a sample text message"
}'
</code>
                    </pre>
                  </v-tab-item>
                  <v-tab-item value="c-sharp">
                    <pre v-highlight class="c-sharp w-full mt-n2 mb-n13">
<code>var client = new HttpClient();
client.DefaultRequestHeaders.Add("x-api-key", ""/* Get API Key from https://httpsms.com/settings */);

var response = await client.PostAsync(
    "https://api.httpsms.com/v1/messages/send",
    new StringContent(
        JsonSerializer.Serialize(new {
            from = "+18005550199",
            To = "+18005550100",
            Content = "This is a sample text message",
        }),
        Encoding.UTF8,
        "application/json"
    )
);

Console.WriteLine(await response.Content.ReadAsStringAsync());
</code>
                    </pre>
                  </v-tab-item>
                </v-tabs-items>
              </div>
            </v-col>
          </v-row>
        </v-col>
      </v-row>
    </v-container>
    <v-footer absolute>
      <v-container class="text-center">
        <v-row justify="center" no-gutters>
          <v-col cols="12">
            <v-btn
              text
              rounded
              outlined
              class="my-2"
              :href="$store.getters.getAppData.documentationUrl"
              >Documentation</v-btn
            >
            <v-btn
              text
              rounded
              class="my-2"
              outlined
              :href="$store.getters.getAppData.githubUrl"
              >Open Source</v-btn
            >
          </v-col>
          <v-col class="text-center mb-n4" cols="12">
            <p class="text--secondary text-subtitle-2">
              &copy; {{ new Date().getFullYear() }} -
              {{ $store.getters.getAppData.name }}
            </p>
          </v-col>
        </v-row>
      </v-container>
    </v-footer>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  mdiGithub,
  mdiCheckCircle,
  mdiSend,
  mdiForum,
  mdiNumeric1,
  mdiNumeric2,
  mdiNumeric3,
  mdiTallyMark1,
  mdiTallyMark3,
  mdiTallyMark2,
  mdiLanguageJavascript,
  mdiLanguagePhp,
  mdiLanguageCsharp,
  mdiLanguageJava,
  mdiPowershell,
  mdiLanguageGo,
} from '@mdi/js'

export default Vue.extend({
  name: 'LandingPage',
  middleware: ['guest'],
  data() {
    return {
      mdiGithub,
      mdiCheckCircle,
      mdiSend,
      mdiForum,
      mdiNumeric1,
      mdiNumeric2,
      mdiNumeric3,
      mdiTallyMark1,
      mdiTallyMark2,
      mdiTallyMark3,
      mdiLanguageJavascript,
      mdiLanguagePhp,
      mdiLanguageCsharp,
      mdiLanguageJava,
      mdiPowershell,
      mdiLanguageGo,
      selectedTab: 'javascript',
    }
  },
})
</script>

<style lang="scss">
.fade-enter-active {
  transition: opacity 0.5s;
}

.fade-enter,
.fade-leave-active {
  opacity: 0;
}

.min-h-650 {
  min-height: 650px;
}

.gradient-header {
  color: #1ad37f;
  background-image: -webkit-linear-gradient(0deg, #1ad37f 14%, #329ef4 55%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}
</style>

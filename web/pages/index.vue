<template>
  <div>
    <v-container>
      <v-row :class="{ 'py-16': $vuetify.breakpoint.lgAndUp }">
        <v-col
          cols="12"
          md="6"
          :class="{
            'text-center pt-16': $vuetify.breakpoint.mdAndDown,
            'py-16': $vuetify.breakpoint.lgAndUp,
          }"
        >
          <h1
            class="text-h2 font-weight-bold pb-1 gradient-header"
            :class="{
              'mt-16': $vuetify.breakpoint.lgAndUp,
              'mt-10': $vuetify.breakpoint.mdOnly,
              'mt-8': $vuetify.breakpoint.smAndDown,
            }"
          >
            Convert your Android phone into an SMS gateway.
          </h1>
          <h2 class="text--secondary text-h5 mt-8 mb-8">
            <span class="gradient-underline">Save money</span> by using your
            phone to send and receive SMS messages via a simple programmable API
            with end-to-end encryption.
          </h2>
          <div :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }">
            <v-btn
              color="primary"
              large
              exact-path
              class="mt-4 mb-4"
              :to="{ name: 'login' }"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp" left>{{
                mdiSend
              }}</v-icon>
              Get Started
            </v-btn>
            <v-btn
              large
              color="secondary"
              class="mt-4 mb-4 ml-4"
              href="https://sandbox.httpsms.com"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp" left color="#ffe500">
                {{ mdiCreation }}
              </v-icon>
              Live Demo
            </v-btn>
          </div>
          <p class="text-body-2 mt-2">
            ⚡Trusted by <b>5,952+</b> happy users who have sent or received
            more than <b>2,860,049+</b> messages.
          </p>
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
        <v-col
          v-if="$vuetify.breakpoint.mdAndUp"
          cols="12"
          md="6"
          class="py-16"
        >
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
              <h3 class="text-h3 mb-1">
                Bulk SMS
                <v-chip class="ma-2" color="pink" label text-color="white">
                  <v-icon left>{{ mdiLabel }}</v-icon>
                  No code
                </v-chip>
              </h3>
              <h5 class="text-h6 font-weight-light">
                Fill in our bulk SMS
                <a
                  class="text-decoration-none"
                  download
                  href="/templates/httpsms-bulk.csv"
                  >CSV template</a
                >
                or our
                <a
                  class="text-decoration-none"
                  download
                  href="/templates/httpsms-bulk.xlsx"
                  >excel template</a
                >
                and upload it on httpSMS to send your SMS messages to multiple
                recipients at once without writing any code.
              </h5>
              <v-btn
                to="/blog/how-to-send-sms-messages-from-excel"
                class="primary mt-4"
              >
                <v-icon left>{{ mdiMicrosoftExcel }}</v-icon>
                Integration Guide
              </v-btn>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="400"
              contain
              :src="require('assets/img/bulk-sms-template.png')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mb-16 mt-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="1">
            <div>
              <h3 class="text-h3 mb-1">
                Integrations
                <v-chip class="ma-2" color="pink" label text-color="white">
                  <v-icon left>{{ mdiLabel }}</v-icon>
                  No code
                </v-chip>
              </h3>
              <h5 class="text-h6 font-weight-light">
                Connect your workflow with thousands of other apps with the
                power of Zapier. For example you can setup an automation to send
                personalized SMS messages each time someone makes an order from
                your shopify store or each time a new row is added to a Google
                spreadsheet.
              </h5>
              <v-btn
                to="/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier"
                class="primary mt-4"
              >
                Zapier Integration Guide
              </v-btn>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="2">
            <v-img
              class="mb-4"
              max-height="400"
              contain
              :src="require('assets/img/zapier-logo.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mb-16 mt-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="2">
            <div>
              <h3 class="text-h3 mb-1">Webhooks</h3>
              <h5 class="text-h6 font-weight-light">
                If you want to build advanced integrations, we support callback
                URLs. The httpSMS platform can forward SMS messages received on
                your Android phone to your server using a callback URL which you
                provide.
              </h5>
              <v-btn
                to="https://docs.httpsms.com/webhooks/introduction"
                class="primary mt-4"
              >
                <v-icon left>{{ mdiWebhook }}</v-icon>
                Documentation
              </v-btn>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/connection.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mb-16 mt-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="1">
            <div>
              <h3 class="text-h3 mb-1">Control Sending</h3>
              <h5 class="text-h6 font-weight-light">
                Send bulk messages without mobile carrier limitations. If you
                set a rate e.g 3 messages per minute, we will queue up your
                messages and send them at a rate of 1 message per 20 seconds.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="2">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/queue.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mb-16 mt-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="2">
            <div>
              <h3 class="text-h3 mb-1">Monitoring</h3>
              <h5 class="text-h6 font-weight-light">
                If your android phone goes offline for some reason and it can't
                send SMS messages, we will send you a notification immediately.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/alert.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mt-16 mb-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="1">
            <div>
              <h3 class="text-h3 mb-1">Open Source</h3>
              <h5 class="text-h6 mb-3 font-weight-light">
                httpSMS is transparent and fully open source. The source code is
                available on GitHub. Feel free to fork it, verify it or submit a
                pull request.
              </h5>
              <a
                class="text-decoration-none"
                :href="$store.getters.getAppData.githubUrl"
              >
                <img
                  alt="GitHub Repo stars"
                  src="https://img.shields.io/github/stars/NdoleStudio/httpsms?style=social"
                />
              </a>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="2">
            <v-img
              class="mb-4"
              max-height="400"
              contain
              :src="require('assets/img/httpsms-github.png')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mt-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="2">
            <div>
              <h3 class="text-h3 mb-1">Encryption 🔐</h3>
              <h5 class="text-h6 mb-3 font-weight-light">
                Take control of your privacy with our end-to-end encrypted SMS
                feature. Safeguard your messages from prying eyes, ensuring
                absolute confidentiality using the military grade
                <a
                  class="text-decoration-none"
                  href="https://en.wikipedia.org/wiki/Advanced_Encryption_Standard"
                  >AES-256 encryption</a
                >
                algorithm.
              </h5>
              <v-btn
                to="/blog/end-to-end-encryption-to-sms-messages"
                class="primary"
                >Setup end-to-end encryption</v-btn
              >
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/mobile-encryption.svg')"
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
      <v-row>
        <v-col cols="12">
          <v-row>
            <v-col cols="12" md="5" class="pr-4">
              <v-timeline dense class="mt-10 ml-n4">
                <v-timeline-item color="primary" :icon="mdiTallyMark1">
                  <v-card class="elevation-2">
                    <v-card-title class="text-h5"> Step 1 </v-card-title>
                    <v-card-text class="subtitle-1">
                      <NuxtLink
                        class="font-weight-bold text-decoration-none"
                        :to="{ name: 'login' }"
                      >
                        Create an account
                      </NuxtLink>
                      on httpsms.com and obtain you API key on the settings
                      page.
                    </v-card-text>
                  </v-card>
                </v-timeline-item>
                <v-timeline-item color="primary" :icon="mdiTallyMark2">
                  <v-card class="elevation-2">
                    <v-card-title class="text-h5"> Step 2 </v-card-title>
                    <v-card-text class="subtitle-1">
                      <a
                        download
                        class="font-weight-bold text-decoration-none"
                        :href="$store.getters.getAppData.appDownloadUrl"
                        @click="
                          $store.dispatch('addNotification', {
                            type: 'info',
                            message: 'Downloading the httpSMS Android App',
                          })
                        "
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
              :class="{ 'pt-16 mb-16mb-16': $vuetify.breakpoint.mdAndUp }"
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
                  <v-tab href="#python">
                    <v-icon color="#ffffff" class="mr-2">{{
                      mdiLanguagePython
                    }}</v-icon>
                    Python
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
<code>import HttpSms from 'httpsms'

const client = new HttpSms('' /* Get the API Key from https://httpsms.com/settings */);

client.messages.postSend({
    content:   'This is a sample text message',
    from:      '+18005550199', // Put the correct phone number here
    to:        '+18005550100', // Put the correct phone number here
})
.then((message) => {
    console.log(message.id); // log the ID of the sent message
})
</code>
                    </pre>
                  </v-tab-item>
                  <v-tab-item value="php">
                    <pre v-highlight class="php w-full mt-n2 mb-n13">
<code>&#60;?php
$apiKey = "Get API Key from https://httpsms.com/settings";

$options = array(
  'http' => array(
    'method'  => 'POST',
    'content' => json_encode( [
        'content' => 'This is a sample text message',
        'from'    => "+18005550199", // Put the correct phone number here
        'to'      => "+18005550100"  // Put the correct phone number here
    ]),
    'header'=>  "Content-Type: application/json\r\n" .
                "Accept: application/json\r\n" .
                "x-api-key: $apiKey\r\n"
    )
);

$context  = stream_context_create( $options );
$result = file_get_contents( "https://api.httpsms.com/v1/messages/send", false, $context );

echo $result;
</code>
                    </pre>
                  </v-tab-item>
                  <v-tab-item value="python">
                    <pre v-highlight class="python w-full mt-n2 mb-n13">
<code>import requests
import json

api_key = "Get API Key from https://httpsms.com/settings"

url = 'https://api.httpsms.com/v1/messages/send'

headers = {
    'x-api-key': api_key,
    'Accept': 'application/json',
    'Content-Type': 'application/json'
}

payload = {
    "content": "This is a sample text message",
    "from": "+18005550199",
    "to": "+18005550100"
}

response = requests.post(url, headers=headers, data=json.dumps(payload))

print(json.dumps(response.json(), indent=4))
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

var payload = """
        {
            "content": "This is a sample text message",
            "from": "+18005550199",
            "to": "+18005550100"
        }
        """;

var request = HttpRequest.newBuilder()
        .uri(URI.create("https://api.httpsms.com/v1/messages/send"))
        .header("accept", "application/json")
        .header("Content-Type", "application/json")
        .header("x-api-key", apiKey)
        .POST(HttpRequest.BodyPublishers.ofString(payload))
        .build();

var response = client.send(request, HttpResponse.BodyHandlers.ofString());
System.out.println(response.body());
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
    <v-sheet class="mt-16 pb-16">
      <v-container>
        <v-row>
          <v-col class="mt-16" md="6" offset-md="3">
            <h2
              id="pricing"
              class="text-center primary--text text-h2 mb-4 text-decoration-underline"
            >
              <span class="text--primary">Pricing</span>
            </h2>
            <h4 class="text-center text-h5 text--secondary">
              Most of the httpSMS features are completely
              <span class="primary--text">free</span> but if you want a little
              extra, you can go pro
            </h4>
            <div class="d-flex justify-center mt-4 align-center">
              <p class="text-h6 mr-3 mt-3">Monthly</p>
              <v-switch v-model="yearlyPricing"></v-switch>
              <p class="text-h6 ml-3 mt-3">
                Yearly
                <v-chip color="primary" small>
                  <v-avatar left>
                    <v-icon small>{{ mdiGift }}</v-icon>
                  </v-avatar>
                  2 months free</v-chip
                >
              </p>
            </div>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" lg="4">
            <v-card elevation="4" color="#121212">
              <v-card-text>
                <h1 class="text-center text-h3 text--primary">Free</h1>
                <p class="subtitle-1 text-center">
                  Try sending and receiving SMS on your hobby websites and
                  experiments.
                </p>
                <p class="text-center text--primary">
                  <span class="text-h3">$0</span>
                </p>
                <p class="text-center mt-n3">No credit card required</p>
                <v-btn block :to="{ name: 'login' }" class="white--text" large
                  >Get Started</v-btn
                >
                <p class="mt-8 subtitle-1">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Send or receive up to <b>200</b> SMS/month
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>
                    {{ mdiCheckCircle }} </v-icon
                  >Offline notifications for your phone
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Forward received messages via webhook
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Basic email support
                </p>
              </v-card-text>
            </v-card>
          </v-col>
          <v-col cols="12" lg="4">
            <v-card elevation="4" color="black">
              <v-card-text>
                <h1 class="text-center text-h3 primary--text">Pro</h1>
                <p class="subtitle-1 text-center">
                  Send and receive more SMS messages like a pro with advanced
                  features.
                </p>
                <p v-if="!yearlyPricing" class="text-center text--primary">
                  <span class="text-h3">$10</span>/month
                </p>
                <p v-else class="text-center text--primary">
                  <span class="text-h3">$100</span>/year
                </p>
                <p v-if="!yearlyPricing" class="text-center mt-n3">
                  or <b>$100</b> per year
                </p>
                <p v-else class="text-center mt-n3">
                  or <b>$8.33</b> per month
                </p>
                <v-btn block color="primary" :to="{ name: 'login' }" large
                  >Try For Free</v-btn
                >
                <p class="mt-8 subtitle-1">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Send or receive up to <b>5,000</b> SMS/month
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>
                    {{ mdiCheckCircle }} </v-icon
                  >Offline notifications for your phone
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Forward received messages via webhook
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Priority support
                </p>
              </v-card-text>
            </v-card>
          </v-col>
          <v-col cols="12" lg="4">
            <v-card elevation="4" color="#121212">
              <v-card-text>
                <h1 class="text-center text-h3 text--primary">Ultra</h1>
                <p class="subtitle-1 text-center">
                  Send and receive up to 10,000 SMS messages like a power user.
                </p>
                <p v-if="!yearlyPricing" class="text-center text--primary">
                  <span class="text-h3">$20</span>/month
                </p>
                <p v-else class="text-center text--primary">
                  <span class="text-h3">$200</span>/year
                </p>
                <p v-if="!yearlyPricing" class="text-center mt-n3">
                  or <b>$200</b> per year
                </p>
                <p v-else class="text-center mt-n3">
                  or <b>$16.66</b> per month
                </p>
                <v-btn block color="default" :to="{ name: 'login' }" large
                  >Try For Free</v-btn
                >
                <p class="mt-8 subtitle-1">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Send or receive up to <b>10,000</b> SMS/month
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>
                    {{ mdiCheckCircle }} </v-icon
                  >Offline notifications for your phone
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Forward received messages via webhook
                </p>
                <p class="subtitle-1 mt-n4">
                  <v-icon color="primary" class="mt-n1" left>{{
                    mdiCheckCircle
                  }}</v-icon
                  >Priority support
                </p>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
        <v-row>
          <v-col>
            <v-alert
              color="info"
              :icon="$vuetify.breakpoint.mdAndUp ? mdiLightbulbOn60 : undefined"
              :prominent="$vuetify.breakpoint.mdAndUp"
              text
            >
              Feel free to <a href="mailto:arnold@httpsms.com">contact us</a> if
              you need a bigger plan, or if you want us to install the httpSMS
              API on your dedicated server. If would still like to support us,
              please donate via
              <a href="https://github.com/sponsors/NdoleStudio"
                >GitHub Sponsors</a
              >
              ❤️
            </v-alert>
          </v-col>
        </v-row>
      </v-container>
    </v-sheet>
    <v-container>
      <v-row class="mt-16">
        <v-col cols="12" md="6">
          <v-card
            href="https://www.g2.com/products/httpsms/reviews/httpsms-review-8589834"
          >
            <v-card-text>
              <div class="d-flex">
                <v-avatar>
                  <v-img
                    alt="Joysankar M."
                    src="https://images.g2crowd.com/uploads/avatar/image/1662077/thumb_square_d5706804d1b343744a8feb693827fe34.jpeg"
                  ></v-img>
                </v-avatar>
                <div>
                  <p class="title ml-3">Joysankar M.</p>
                  <v-rating
                    class="mt-n7"
                    color="yellow darken-3"
                    :value="4.5"
                    half-increments
                  ></v-rating>
                </div>
                <v-spacer />
                <div style="width: 30px">
                  <v-img
                    max-height="30"
                    contain
                    src="https://company.g2.com/hs-fs/hubfs/brand-guide/reversed-g2@2x.png"
                  />
                </div>
              </div>
              <p class="blockquote mb-0">
                "httpSMS is free platform which transforms your phone into an
                sms server! It has no hard limit also. It is an
                <b>innovative</b> idea, I have not seen such tech before. If you
                have an <b>sms active pack</b> in your phone then good to go
                with httpSMS."
              </p>
            </v-card-text>
          </v-card>
        </v-col>
        <v-col cols="12" md="6">
          <v-card href="https://www.uneed.best/tool/httpsmscom?tab=comments">
            <v-card-text>
              <div class="d-flex">
                <v-avatar>
                  <v-img
                    alt="Edmund Ciego Profile Picture"
                    src="https://lh3.googleusercontent.com/a/ACg8ocJktUViyMcJvzkPNpza7SZ3ql_nwOAzYk0uJ27TF5L_z0bRoPKE=s96-c"
                  ></v-img>
                </v-avatar>
                <div>
                  <p class="title ml-3">Edmund Ciego</p>
                  <v-rating
                    class="mt-n7"
                    color="yellow darken-3"
                    :value="5"
                    half-increments
                  ></v-rating>
                </div>
                <v-spacer />
                <div>
                  <v-img
                    contain
                    max-width="64"
                    :src="require('@/assets/img/logos/uneed.svg')"
                  />
                </div>
              </div>
              <p class="blockquote mb-0">
                "<b>Outstanding product</b>. Literally have been using this for
                years since we don't have an sms gateways that can handle http
                requests costing less than <b>50 cent per sms</b> in my Country.
                Love the product and the support! Great work Arnold!"
              </p>
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>
      <v-row class="mt-16">
        <v-col md="8" offset-md="2">
          <h2 class="text-h3 text-center">Frequently Asked Questions</h2>
          <h4 class="text-center text-h6 mt-4 text--secondary">
            If you still cannot find the answer to your question,
            <a href="mailto:arnold@httpsms.com">send us an email</a> or ask in
            our <a href="https://discord.gg/kGk8HVqeEZ">Discord</a> channel.
          </h4>
        </v-col>
      </v-row>
      <v-row>
        <v-col md="8" offset-md="2" class="mb-16">
          <v-expansion-panels v-model="faqPanel" focusable>
            <v-expansion-panel>
              <v-expansion-panel-header class="text-h6">
                Can I install the app on my Iphone?
                <template v-slot:actions>
                  <v-icon v-if="faqPanel === 0">{{ mdiMinus }}</v-icon>
                  <v-icon v-else>{{ mdiPlus }}</v-icon>
                </template>
              </v-expansion-panel-header>
              <v-expansion-panel-content>
                <p class="mt-8">
                  The httpSMS application works only on Android phones at the
                  moment since Apple doesn't allow you to install a custom SMS
                  messaging app.
                </p>
              </v-expansion-panel-content>
            </v-expansion-panel>
            <v-expansion-panel>
              <v-expansion-panel-header class="text-h6">
                What's the minimum supported Android version?
                <template v-slot:actions>
                  <v-icon v-if="faqPanel === 1">{{ mdiMinus }}</v-icon>
                  <v-icon v-else>{{ mdiPlus }}</v-icon>
                </template>
              </v-expansion-panel-header>
              <v-expansion-panel-content>
                <p class="mt-8">
                  The httpSMS Android app works from Android 9 (Pie) and above.
                  So you can install the application on your old Android phone
                  which you don't use anymore.
                </p>
              </v-expansion-panel-content>
            </v-expansion-panel>
            <v-expansion-panel>
              <v-expansion-panel-header class="text-h6">
                Can I send unlimited number of messages per month?
                <template v-slot:actions>
                  <v-icon v-if="faqPanel === 2">{{ mdiMinus }}</v-icon>
                  <v-icon v-else>{{ mdiPlus }}</v-icon>
                </template>
              </v-expansion-panel-header>
              <v-expansion-panel-content>
                <p class="mt-8">
                  We do have packages that allow up to 100,000 SMS messages per
                  month but you can can
                  <a href="mailto:arnold@httpsms.com">send us an email</a> if
                  you will like to send more messages so we create a custom plan
                  just for you.
                </p>
              </v-expansion-panel-content>
            </v-expansion-panel>
            <v-expansion-panel>
              <v-expansion-panel-header class="text-h6">
                Can I change the sender of the SMS message
                <template v-slot:actions>
                  <v-icon v-if="faqPanel === 3">{{ mdiMinus }}</v-icon>
                  <v-icon v-else>{{ mdiPlus }}</v-icon>
                </template>
              </v-expansion-panel-header>
              <v-expansion-panel-content>
                <p class="mt-8">
                  No you cannot. When you send an SMS message using the httpSMS
                  app it uses your SIM card to send the message so the recipient
                  will see your phone number as the sender of the SMS. You
                  cannot use your brand name as the sender ID.
                </p>
              </v-expansion-panel-content>
            </v-expansion-panel>
          </v-expansion-panels>
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  mdiGithub,
  mdiCheckCircle,
  mdiSend,
  mdiGift,
  mdiLightbulbOn60,
  mdiForum,
  mdiCreation,
  mdiNumeric1,
  mdiSale,
  mdiLanguagePython,
  mdiNumeric2,
  mdiNumeric3,
  mdiTallyMark1,
  mdiTallyMark3,
  mdiTallyMark2,
  mdiLabel,
  mdiLanguageJavascript,
  mdiLanguagePhp,
  mdiPlus,
  mdiMinus,
  mdiLanguageCsharp,
  mdiLanguageJava,
  mdiMicrosoftExcel,
  mdiWebhook,
  mdiPowershell,
  mdiLanguageGo,
} from '@mdi/js'

export default Vue.extend({
  name: 'LandingPage',
  layout: 'website',
  data() {
    return {
      mdiMicrosoftExcel,
      mdiWebhook,
      mdiGithub,
      mdiLabel,
      mdiLightbulbOn60,
      mdiCheckCircle,
      mdiSend,
      mdiGift,
      mdiCreation,
      mdiForum,
      mdiNumeric1,
      mdiNumeric2,
      mdiNumeric3,
      mdiTallyMark1,
      mdiSale,
      mdiTallyMark2,
      mdiPlus,
      mdiMinus,
      mdiTallyMark3,
      mdiLanguageJavascript,
      mdiLanguagePhp,
      mdiLanguagePython,
      mdiLanguageCsharp,
      mdiLanguageJava,
      mdiPowershell,
      mdiLanguageGo,
      selectedTab: 'javascript',
      yearlyPricing: false,
      faqPanel: null,
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

.gradient-underline {
  color: white;
}
</style>

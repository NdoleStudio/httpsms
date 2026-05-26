<script setup lang="ts">
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
  mdiCellphoneKey,
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
  mdiClockOutline,
  mdiArrowRightThin,
  mdiPowershell,
  mdiLanguageGo,
} from "@mdi/js";

definePageMeta({
  layout: "website",
});

useHead({
  title: "Convert your android phone into an SMS gateway - httpSMS",
});

const config = useRuntimeConfig();
const { lgAndUp, mdAndUp, mdAndDown, mdOnly, smAndDown, xl } = useDisplay();

const selectedTab = ref("javascript");
const yearlyPricing = ref(false);
const faqPanel = ref<number | undefined>(undefined);
const pricing = ref(0);

const pricingLabels = ["10K", "20K", "50K", "100K", "200K"];
const pricingLabelsFull = ["10,000", "20,000", "50,000", "100,000", "200,000"];

const planMessages = computed(() =>
  pricingLabels[pricing.value].replace("K", ",000"),
);
const planMonthlyPrice = computed(() => [20, 35, 89, 175, 350][pricing.value]);
const planYearlyPrice = computed(
  () => [200, 350, 1068, 2100, 4200][pricing.value],
);
const planYearlyMonthlyPrice = computed(
  () => [16.66, 29.16, 89, 175, 350][pricing.value],
);
</script>

<template>
  <div>
    <VContainer>
      <VRow :class="{ 'py-16': lgAndUp }">
        <VCol
          cols="12"
          md="6"
          :class="{
            'text-center pt-16': mdAndDown,
            'py-16': lgAndUp,
          }"
        >
          <h1
            class="text-display-medium font-weight-bold pb-1 gradient-header"
            :class="{
              'mt-16': lgAndUp,
              'mt-10': mdOnly,
              'mt-8': smAndDown,
            }"
          >
            Convert your Android phone into an SMS gateway.
          </h1>
          <h2 class="text-medium-emphasis text-headline-medium mt-8 mb-8">
            <span class="gradient-underline">Save money</span> by using your
            phone to send and receive SMS messages via a simple programmable API
            with end-to-end encryption.
          </h2>
          <div :class="{ 'text-center': mdAndDown }">
            <VBtn color="primary" size="large" class="mt-4 mb-4" to="/login">
              <VIcon v-if="lgAndUp" start :icon="mdiSend" />
              Get Started
            </VBtn>
            <VBtn
              size="large"
              color="secondary"
              class="mt-4 mb-4 ml-4"
              href="https://sandbox.httpsms.com"
            >
              <VIcon v-if="lgAndUp" start :icon="mdiCreation" color="#ffe500" />
              Live Demo
            </VBtn>
          </div>
          <p class="text-body-medium mt-2">
            ✅Trusted by <b>23,273+</b> users who send/receive more than
            <b>500,000</b> messages per month.
          </p>
          <div class="mt-4" :class="{ 'text-center': mdAndDown }">
            <VIcon color="success" :icon="mdiCheckCircle" />
            Free to use
            <VIcon class="ml-4" color="success" :icon="mdiCheckCircle" />
            100% Open Source
          </div>
          <div v-if="xl" class="mt-4">
            <a href="https://www.uneed.best/tool/httpsmscom">
              <img
                src="https://www.uneed.best/POTD1A.png"
                style="width: 250px"
                alt="Uneed POTD1 Badge"
              />
            </a>
          </div>
          <VDivider
            v-if="mdAndDown"
            class="mt-6 mr-16 bg-success"
            :class="{ 'ml-16': mdAndDown }"
          />
        </VCol>
        <VCol v-if="mdAndUp" cols="12" md="6" class="py-16">
          <VImg
            class="mx-auto mb-4"
            max-height="700"
            max-width="98%"
            :src="'/img/writing-code-phone.svg'"
          />
        </VCol>
      </VRow>
    </VContainer>

    <!-- Features Section -->
    <VSheet class="py-16">
      <VContainer>
        <!-- Bulk SMS -->
        <VRow class="mb-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="2">
            <div>
              <h3 class="text-display-small mb-1">
                Bulk SMS
                <VChip class="ma-2" color="pink" label>
                  <VIcon start :icon="mdiLabel" />
                  No code
                </VChip>
              </h3>
              <h5 class="text-headline-small font-weight-light">
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
                and upload it on httpSMS to send SMS messages to up to 1,000
                recipients at once without writing any code.
              </h5>
              <VBtn
                to="/blog/how-to-send-sms-messages-from-excel"
                color="primary"
                class="mt-4"
              >
                <VIcon start :icon="mdiMicrosoftExcel" />
                Integration Guide
              </VBtn>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="1">
            <VImg
              class="mb-4"
              max-height="400"
              :src="'/img/bulk-sms-template.png'"
            />
          </VCol>
        </VRow>

        <!-- Integrations -->
        <VRow class="mb-16 mt-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="1">
            <div>
              <h3 class="text-display-small mb-1">
                Integrations
                <VChip class="ma-2" color="pink" label>
                  <VIcon start :icon="mdiLabel" />
                  No code
                </VChip>
              </h3>
              <h5 class="text-headline-small font-weight-light">
                Connect your workflow with thousands of other apps with the
                power of Zapier. For example you can setup an automation to send
                personalized SMS messages each time someone makes an order from
                your shopify store or each time a new row is added to a Google
                spreadsheet.
              </h5>
              <VBtn
                to="/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier"
                color="primary"
                class="mt-4"
              >
                Zapier Integration Guide
              </VBtn>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="2">
            <VImg class="mb-4" max-height="400" :src="'/img/zapier-logo.svg'" />
          </VCol>
        </VRow>

        <!-- Webhooks -->
        <VRow class="mb-16 mt-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="2">
            <div>
              <h3 class="text-display-small mb-1">Webhooks</h3>
              <h5 class="text-headline-small font-weight-light">
                If you want to build advanced integrations, we support callback
                URLs. The httpSMS platform can forward SMS messages received on
                your Android phone to your server using a callback URL which you
                provide.
              </h5>
              <VBtn
                target="_blank"
                href="https://docs.httpsms.com/webhooks/introduction"
                color="primary"
                class="mt-4"
              >
                <VIcon start :icon="mdiWebhook" />
                Documentation
              </VBtn>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="1">
            <VImg class="mb-4" max-height="300" :src="'/img/connection.svg'" />
          </VCol>
        </VRow>

        <!-- Control Sending -->
        <VRow class="mb-16 mt-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="1">
            <div>
              <h3 class="text-display-small mb-1">Control Sending</h3>
              <h5 class="text-headline-small font-weight-light">
                Send SMS messages without going over your mobile carrier
                limitations. If you set a rate e.g 3 messages per minute, we
                will queue up your messages and send them at a rate of 1 message
                per 20 seconds.
              </h5>
              <VBtn
                href="https://docs.httpsms.com/features/control-sms-send-rate"
                color="primary"
              >
                <VIcon start :icon="mdiArrowRightThin" />
                Documentation
              </VBtn>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="2">
            <VImg class="mb-4" max-height="300" :src="'/img/queue.svg'" />
          </VCol>
        </VRow>

        <!-- Monitoring -->
        <VRow class="mb-16 mt-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="2">
            <div>
              <h3 class="text-display-small mb-1">Monitoring</h3>
              <h5 class="text-headline-small font-weight-light">
                If your android phone goes offline for some reason and it can't
                send SMS messages, we will send you a notification immediately.
              </h5>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="1">
            <VImg class="mb-4" max-height="300" :src="'/img/alert.svg'" />
          </VCol>
        </VRow>

        <!-- Open Source -->
        <VRow class="mt-16 mb-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="1">
            <div>
              <h3 class="text-display-small mb-1">Open Source</h3>
              <h5 class="text-headline-small mb-3 font-weight-light">
                httpSMS is transparent and fully open source. The source code is
                available on GitHub. Feel free to fork it, verify it or submit a
                pull request.
              </h5>
              <a
                class="text-decoration-none"
                :href="config.public.appGithubUrl"
              >
                <img
                  alt="GitHub Repo stars"
                  src="https://img.shields.io/github/stars/NdoleStudio/httpsms?style=social"
                />
              </a>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="2">
            <VImg
              class="mb-4"
              max-height="400"
              :src="'/img/httpsms-github.png'"
            />
          </VCol>
        </VRow>

        <!-- Encryption -->
        <VRow class="mt-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="2">
            <div>
              <h3 class="text-display-small mb-1">Encryption 🔐</h3>
              <h5 class="text-headline-small mb-3 font-weight-light">
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
              <VBtn
                to="/blog/end-to-end-encryption-to-sms-messages"
                color="primary"
              >
                Setup end-to-end encryption
              </VBtn>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="1">
            <VImg
              class="mb-4"
              max-height="300"
              :src="'/img/mobile-encryption.svg'"
            />
          </VCol>
        </VRow>

        <!-- Multiple Phones -->
        <VRow class="mt-16">
          <VCol cols="12" md="6" class="d-flex align-center">
            <div>
              <h3 class="text-display-small mb-1">Multiple Phones</h3>
              <h5 class="text-headline-small mb-3 font-weight-light">
                Setup the httpSMS gateway Android app on multiple phones
                independently and securely without sharing data under one
                account by creating unique phone API keys.
              </h5>
              <VBtn
                href="https://docs.httpsms.com/features/phone-api-keys"
                color="primary"
              >
                <VIcon start :icon="mdiCellphoneKey" />
                Documentation
              </VBtn>
            </div>
          </VCol>
          <VCol cols="12" md="6">
            <VImg
              class="mb-4"
              max-height="300"
              :src="'/img/manage-phones.svg'"
            />
          </VCol>
        </VRow>

        <!-- Schedule Messages -->
        <VRow class="mt-16">
          <VCol cols="12" md="6" class="d-flex align-center" order-lg="2">
            <div>
              <h3 class="text-display-small mb-1">Schedule Text Messages</h3>
              <h5 class="text-headline-small mb-3 font-weight-light">
                Control when your SMS will reach your recipients, allowing you
                to perfectly time promotions, critical alerts etc by scheduling
                your messages in advance.
              </h5>
              <VBtn
                href="https://docs.httpsms.com/features/scheduling-sms-messages"
                color="primary"
              >
                <VIcon start :icon="mdiClockOutline" />
                Documentation
              </VBtn>
            </div>
          </VCol>
          <VCol cols="12" md="6" order-lg="1">
            <VImg
              class="mb-4"
              max-height="300"
              :src="'/img/schedule-messages.svg'"
            />
          </VCol>
        </VRow>
      </VContainer>
    </VSheet>

    <!-- Get Started Section -->
    <VContainer class="pb-16">
      <VRow class="mt-16">
        <VCol>
          <h2 class="text-display-small text-center">Get Started</h2>
        </VCol>
      </VRow>
      <VRow>
        <VCol cols="12">
          <VRow>
            <VCol cols="12" md="5" class="pr-4">
              <VTimeline density="compact" class="mt-10 ml-n4">
                <VTimelineItem dot-color="primary" :icon="mdiTallyMark1">
                  <VCard variant="elevated">
                    <VCardTitle class="text-headline-medium">Step 1</VCardTitle>
                    <VCardText class="text-title-large">
                      <NuxtLink
                        class="font-weight-bold text-decoration-none"
                        to="/login"
                      >
                        Create an account
                      </NuxtLink>
                      on httpsms.com and obtain your API key on the settings
                      page.
                    </VCardText>
                  </VCard>
                </VTimelineItem>
                <VTimelineItem dot-color="primary" :icon="mdiTallyMark2">
                  <VCard variant="elevated">
                    <VCardTitle class="text-headline-medium">Step 2</VCardTitle>
                    <VCardText class="text-title-large">
                      <a
                        download
                        class="font-weight-bold text-decoration-none"
                        :href="config.public.appDownloadUrl"
                        >Download</a
                      >
                      and install the companion android application on your
                      phone and sign in using your API Key.
                    </VCardText>
                  </VCard>
                </VTimelineItem>
                <VTimelineItem dot-color="primary" :icon="mdiTallyMark3">
                  <VCard variant="elevated">
                    <VCardTitle class="text-headline-medium">Step 3</VCardTitle>
                    <VCardText class="text-title-large">
                      Start sending and receiving SMS messages using our rich
                      HTTP API. You can find the documentation on
                      <a
                        class="text-decoration-none"
                        :href="config.public.appDocumentationUrl"
                      >
                        {{ config.public.appDocumentationUrl }}
                      </a>
                    </VCardText>
                  </VCard>
                </VTimelineItem>
              </VTimeline>
            </VCol>
            <VCol cols="12" md="7" :class="{ 'pt-16 mb-16': mdAndUp }">
              <div class="w-100 mt-4">
                <VTabs v-model="selectedTab" show-arrows>
                  <VTab value="javascript">
                    <VIcon
                      color="#efd81d"
                      class="mr-1"
                      :icon="mdiLanguageJavascript"
                    />
                    Javascript
                  </VTab>
                  <VTab value="php">
                    <VIcon
                      color="#777bb3"
                      class="mr-2"
                      :icon="mdiLanguagePhp"
                    />
                    PHP
                  </VTab>
                  <VTab value="python">
                    <VIcon
                      color="#ffffff"
                      class="mr-2"
                      :icon="mdiLanguagePython"
                    />
                    Python
                  </VTab>
                  <VTab value="go">
                    <VIcon color="#00aed8" class="mr-2" :icon="mdiLanguageGo" />
                    Go
                  </VTab>
                  <VTab value="java">
                    <VIcon
                      color="#0c89c7"
                      class="mr-2"
                      :icon="mdiLanguageJava"
                    />
                    Java
                  </VTab>
                  <VTab value="curl">
                    <VIcon color="primary" class="mr-2" :icon="mdiPowershell" />
                    cURL
                  </VTab>
                  <VTab value="c-sharp">
                    <VIcon
                      color="#68217a"
                      class="mr-2"
                      :icon="mdiLanguageCsharp"
                    />
                    C#
                  </VTab>
                </VTabs>
                <VTabsWindow v-model="selectedTab">
                  <VTabsWindowItem value="javascript">
                    <pre
                      class="pa-4 bg-surface-variant rounded mt-2"
                    ><code>import HttpSms from 'httpsms'

const client = new HttpSms('' /* Get the API Key from https://httpsms.com/settings */);

client.messages.postSend({
    content:   'This is a sample text message',
    from:      '+18005550199', // Put the correct phone number here
    to:        '+18005550100', // Put the correct phone number here
})
.then((message) => {
    console.log(message.id); // log the ID of the sent message
})</code></pre>
                  </VTabsWindowItem>
                  <VTabsWindowItem value="php">
                    <pre
                      class="pa-4 bg-surface-variant rounded mt-2"
                    ><code>&lt;?php
$apiKey = "Get API Key from https://httpsms.com/settings";

$options = array(
  'http' => array(
    'method'  => 'POST',
    'content' => json_encode( [
        'content' => 'This is a sample text message',
        'from'    => "+18005550199",
        'to'      => "+18005550100"
    ]),
    'header'=>  "Content-Type: application/json\r\n" .
                "Accept: application/json\r\n" .
                "x-api-key: $apiKey\r\n"
    )
);

$context  = stream_context_create( $options );
$result = file_get_contents( "https://api.httpsms.com/v1/messages/send", false, $context );

echo $result;</code></pre>
                  </VTabsWindowItem>
                  <VTabsWindowItem value="python">
                    <pre
                      class="pa-4 bg-surface-variant rounded mt-2"
                    ><code>import requests
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

print(json.dumps(response.json(), indent=4))</code></pre>
                  </VTabsWindowItem>
                  <VTabsWindowItem value="go">
                    <pre
                      class="pa-4 bg-surface-variant rounded mt-2"
                    ><code>import "github.com/NdoleStudio/httpsms-go"

client := htpsms.New(htpsms.WithAPIKey(/* API Key from https://httpsms.com/settings */))

client.Messages.Send(context.Background(), &amp;httpsms.MessageSendParams{
    Content: "This is a sample text message",
    From:    "+18005550199",
    To:      "+18005550100",
})</code></pre>
                  </VTabsWindowItem>
                  <VTabsWindowItem value="java">
                    <pre
                      class="pa-4 bg-surface-variant rounded mt-2"
                    ><code>var client = HttpClient.newHttpClient();
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
System.out.println(response.body());</code></pre>
                  </VTabsWindowItem>
                  <VTabsWindowItem value="curl">
                    <pre
                      class="pa-4 bg-surface-variant rounded mt-2"
                    ><code>curl --location --request POST 'https://api.httpsms.com/v1/messages/send' \
--header 'x-api-key: Get API Key from https://httpsms.com/settings' \
--header 'Content-Type: application/json' \
--data-raw '{
    "from": "+18005550199",
    "to": "+18005550100",
    "content": "This is a sample text message"
}'</code></pre>
                  </VTabsWindowItem>
                  <VTabsWindowItem value="c-sharp">
                    <pre
                      class="pa-4 bg-surface-variant rounded mt-2"
                    ><code>var client = new HttpClient();
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

Console.WriteLine(await response.Content.ReadAsStringAsync());</code></pre>
                  </VTabsWindowItem>
                </VTabsWindow>
              </div>
            </VCol>
          </VRow>
        </VCol>
      </VRow>
    </VContainer>

    <!-- Pricing Section -->
    <VSheet class="mt-16 pb-16">
      <VContainer>
        <VRow>
          <VCol class="mt-16" md="6" offset-md="3">
            <h2
              id="pricing"
              class="text-center text-display-medium mb-4 text-decoration-underline text-primary"
            >
              Pricing
            </h2>
            <h4 class="text-center text-headline-medium text-medium-emphasis">
              Most of the httpSMS features are completely
              <span class="text-primary">free</span> but if you want a little
              extra, you can go pro
            </h4>
            <div class="d-flex justify-center mt-4 align-center">
              <p class="text-headline-small mr-3 mt-3">Monthly</p>
              <VSwitch v-model="yearlyPricing" hide-details />
              <p class="text-headline-small ml-3 mt-3">
                Yearly
                <VChip color="primary" size="small">
                  <VIcon start :icon="mdiGift" size="small" />
                  2 months free
                </VChip>
              </p>
            </div>
          </VCol>
        </VRow>
        <VRow>
          <VCol cols="12">
            <VSlider
              v-model="pricing"
              :tick-labels="lgAndUp ? pricingLabelsFull : pricingLabels"
              :max="4"
              step="1"
              thumb-label="always"
              thumb-size="32"
              show-ticks="always"
            >
              <template #thumb-label>
                {{ pricingLabels[pricing] }}
              </template>
            </VSlider>
          </VCol>
        </VRow>
        <VRow>
          <!-- Free Plan -->
          <VCol cols="12" lg="4">
            <VCard elevation="4" color="#121212">
              <VCardText>
                <h1 class="text-center text-display-small">Free</h1>
                <p class="text-title-large text-center">
                  Try sending and receiving SMS on your hobby websites and
                  experiments.
                </p>
                <p class="text-center">
                  <span class="text-display-small">$0</span>
                </p>
                <p class="text-center mt-n3">No credit card required</p>
                <VBtn block to="/login" size="large">Get Started</VBtn>
                <p class="mt-8 text-title-large">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Send or receive up to <b>200</b> SMS/month
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Offline notifications for your phone
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Forward received messages via webhook
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Basic email support
                </p>
              </VCardText>
            </VCard>
          </VCol>
          <!-- Pro Plan -->
          <VCol cols="12" lg="4">
            <VCard elevation="4" color="black">
              <VCardText>
                <h1 class="text-center text-display-small text-primary">Pro</h1>
                <p class="text-title-large text-center">
                  Send and receive more SMS messages like a pro with advanced
                  features.
                </p>
                <p v-if="!yearlyPricing" class="text-center">
                  <span class="text-display-small">$10</span>/month
                </p>
                <p v-else class="text-center">
                  <span class="text-display-small">$100</span>/year
                </p>
                <p v-if="!yearlyPricing" class="text-center mt-n3">
                  or <b>$100</b> per year
                </p>
                <p v-else class="text-center mt-n3">
                  or <b>$8.33</b> per month
                </p>
                <VBtn block color="primary" to="/login" size="large"
                  >Try For Free</VBtn
                >
                <p class="mt-8 text-title-large">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Send or receive up to <b>5,000</b> SMS/month
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Offline notifications for your phone
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Forward received messages via webhook
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Priority support
                </p>
              </VCardText>
            </VCard>
          </VCol>
          <!-- Custom Plan -->
          <VCol cols="12" lg="4">
            <VCard color="#121212" variant="outlined">
              <VCardText>
                <h1 class="text-center text-display-small">
                  {{ pricingLabels[pricing] }} Plan
                </h1>
                <p class="text-title-large text-center">
                  Send and receive up to {{ planMessages }} SMS messages like a
                  power user.
                </p>
                <p v-if="!yearlyPricing" class="text-center">
                  <span class="text-display-small">${{ planMonthlyPrice }}</span
                  >/month
                </p>
                <p v-else class="text-center">
                  <span class="text-display-small">${{ planYearlyPrice }}</span
                  >/year
                </p>
                <p v-if="!yearlyPricing" class="text-center mt-n3">
                  or <b>${{ planYearlyPrice }}</b> per year
                </p>
                <p v-else class="text-center mt-n3">
                  or <b>${{ planYearlyMonthlyPrice }}</b> per month
                </p>
                <VBtn block variant="tonal" to="/login" size="large"
                  >Try For Free</VBtn
                >
                <p class="mt-8 text-title-large">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Send or receive up to
                  <b>{{ pricingLabels[pricing] }}</b> SMS/month
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Offline notifications for your phone
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Forward received messages via webhook
                </p>
                <p class="text-title-large mt-n4">
                  <VIcon
                    color="primary"
                    class="mt-n1"
                    start
                    :icon="mdiCheckCircle"
                  />
                  Priority support
                </p>
              </VCardText>
            </VCard>
          </VCol>
        </VRow>
        <VRow>
          <VCol>
            <VAlert
              color="info"
              :icon="mdAndUp ? mdiLightbulbOn60 : undefined"
              :prominent="mdAndUp"
              variant="tonal"
            >
              Feel free to <a href="mailto:arnold@httpsms.com">contact us</a> if
              you need a bigger plan, or if you want us to install the httpSMS
              API on your dedicated server. If you would still like to support
              us, please donate via
              <a href="https://github.com/sponsors/NdoleStudio"
                >GitHub Sponsors</a
              >.
            </VAlert>
          </VCol>
        </VRow>
      </VContainer>
    </VSheet>

    <!-- FAQ Section -->
    <VContainer class="pb-16 mt-16">
      <VRow>
        <VCol md="8" offset-md="2" class="mb-16">
          <VExpansionPanels v-model="faqPanel">
            <VExpansionPanel>
              <VExpansionPanelTitle class="text-headline-small">
                Can I install the app on my iPhone?
                <template #actions>
                  <VIcon :icon="faqPanel === 0 ? mdiMinus : mdiPlus" />
                </template>
              </VExpansionPanelTitle>
              <VExpansionPanelText>
                <p class="mt-4">
                  The httpSMS application works only on Android phones at the
                  moment since Apple doesn't allow you to install a custom SMS
                  messaging app.
                </p>
              </VExpansionPanelText>
            </VExpansionPanel>
            <VExpansionPanel>
              <VExpansionPanelTitle class="text-headline-small">
                What's the minimum supported Android version?
                <template #actions>
                  <VIcon :icon="faqPanel === 1 ? mdiMinus : mdiPlus" />
                </template>
              </VExpansionPanelTitle>
              <VExpansionPanelText>
                <p class="mt-4">
                  The httpSMS Android app works from Android 9 (Pie) and above.
                  So you can install the application on your old Android phone
                  which you don't use anymore.
                </p>
              </VExpansionPanelText>
            </VExpansionPanel>
            <VExpansionPanel>
              <VExpansionPanelTitle class="text-headline-small">
                Can I send unlimited number of messages per month?
                <template #actions>
                  <VIcon :icon="faqPanel === 2 ? mdiMinus : mdiPlus" />
                </template>
              </VExpansionPanelTitle>
              <VExpansionPanelText>
                <p class="mt-4">
                  We do have packages that allow up to 100,000 SMS messages per
                  month but you can
                  <a href="mailto:arnold@httpsms.com">send us an email</a> if
                  you will like to send more messages so we create a custom plan
                  just for you.
                </p>
              </VExpansionPanelText>
            </VExpansionPanel>
            <VExpansionPanel>
              <VExpansionPanelTitle class="text-headline-small">
                Can I change the sender of the SMS message?
                <template #actions>
                  <VIcon :icon="faqPanel === 3 ? mdiMinus : mdiPlus" />
                </template>
              </VExpansionPanelTitle>
              <VExpansionPanelText>
                <p class="mt-4">
                  No you cannot. When you send an SMS message using the httpSMS
                  app it uses your SIM card to send the message so the recipient
                  will see your phone number as the sender of the SMS. You
                  cannot use your brand name as the sender ID.
                </p>
              </VExpansionPanelText>
            </VExpansionPanel>
          </VExpansionPanels>
        </VCol>
      </VRow>
    </VContainer>
  </div>
</template>

<style lang="scss">
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

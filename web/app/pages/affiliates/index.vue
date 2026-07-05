<script setup lang="ts">
import {
  mdiCashMultiple,
  mdiRepeatVariant,
  mdiMagnet,
  mdiCreditCardCheckOutline,
  mdiArrowRightThin,
  mdiChevronDown,
} from '@mdi/js'

definePageMeta({
  layout: 'website',
})

useSeoMeta({
  title: 'Affiliate Program — Earn up to $70 per Sale | httpSMS',
  description:
    'Join the httpSMS affiliate program and earn up to $70 for every customer you refer, with recurring commissions. Free to join, powered by LemonSqueezy.',
  ogTitle: 'Get paid to share httpSMS — earn up to $70 per sale',
  ogDescription:
    'Join the httpSMS affiliate program and earn up to $70 for every customer you refer, with recurring commissions. Free to join, powered by LemonSqueezy.',
  ogImage: 'https://httpsms.com/header.png',
  twitterCard: 'summary_large_image',
})

const { mdAndUp, mdAndDown } = useDisplay()

const signupUrl = 'https://affiliates.lemonsqueezy.com/programs/httpsms'

const benefits = [
  {
    icon: mdiCashMultiple,
    title: 'Up to $70 per sale',
    body: 'Real commission on real subscriptions, not pennies per click.',
  },
  {
    icon: mdiRepeatVariant,
    title: 'Recurring commission',
    body: 'Get paid every month your referral stays, not just once.',
  },
  {
    icon: mdiMagnet,
    title: 'A product that sticks',
    body: 'Developers wire httpSMS into daily workflows, so they rarely leave.',
  },
  {
    icon: mdiCreditCardCheckOutline,
    title: 'Payouts on autopilot',
    body: 'LemonSqueezy tracks every referral and pays you automatically.',
  },
]

const steps = [
  {
    title: 'Sign up free',
    body: 'Join through LemonSqueezy in under a minute. No cost, no catch.',
  },
  {
    title: 'Share your link',
    body: 'Drop it in your blog posts, videos, docs, or DMs.',
  },
  {
    title: 'Get paid',
    body: 'We handle tracking and payouts. You collect on every sale.',
  },
]

const faqs = [
  {
    question: 'How much can I earn?',
    answer:
      'Up to $70 per sale, plus recurring commission for as long as your referral stays subscribed.',
  },
  {
    question: 'How do I get paid?',
    answer:
      'LemonSqueezy tracks your referrals and pays you automatically on their schedule.',
  },
  {
    question: 'Who can join?',
    answer:
      "Anyone. Bloggers, YouTubers, developers, agencies — and it's free to join.",
  },
  {
    question: 'How do I track my referrals?',
    answer:
      'Your LemonSqueezy dashboard shows clicks, referrals, and earnings in real time.',
  },
  {
    question: 'What converts best?',
    answer:
      'Tutorials, honest reviews, and comparisons that show how httpSMS turns an Android phone into an SMS gateway.',
  },
]

const scrollToHowItWorks = () => {
  document
    .getElementById('how-it-works')
    ?.scrollIntoView({ behavior: 'smooth' })
}
</script>

<template>
  <div>
    <!-- Hero -->
    <VContainer>
      <VRow>
        <VCol cols="12" md="8" offset-md="2" class="pt-16 pb-8 text-center">
          <p
            class="text-overline text-medium-emphasis mb-2"
            style="letter-spacing: 0.3em !important"
          >
            Affiliate Program
          </p>
          <h1 class="text-display-medium font-weight-bold gradient-header mb-6">
            Get paid to share httpSMS
          </h1>
          <h2
            class="text-headline-small font-weight-light text-medium-emphasis mb-8"
          >
            Earn up to <span class="text-white font-weight-bold">$70</span> for
            every customer you refer — and keep earning every month they stay
            subscribed. If you already tell people to ditch expensive short
            codes, you might as well get paid for it.
          </h2>
          <div>
            <VBtn
              color="primary"
              size="large"
              class="ma-2"
              :href="signupUrl"
              target="_blank"
              rel="noopener"
            >
              <VIcon start :icon="mdiCashMultiple" />
              Become an affiliate
            </VBtn>
            <VBtn
              size="large"
              variant="tonal"
              class="ma-2"
              @click="scrollToHowItWorks"
            >
              See how it works
              <VIcon end :icon="mdiChevronDown" />
            </VBtn>
          </div>
        </VCol>
      </VRow>
    </VContainer>

    <!-- Why promote httpSMS -->
    <VSheet class="py-16">
      <VContainer>
        <h2 class="text-headline-large font-weight-bold text-center mb-2">
          Why promote httpSMS
        </h2>
        <p class="text-body-large text-medium-emphasis text-center mb-10">
          A generous program built on a product people actually keep.
        </p>
        <VRow>
          <VCol
            v-for="benefit in benefits"
            :key="benefit.title"
            cols="12"
            sm="6"
            md="3"
          >
            <VCard height="100%" class="pa-4" variant="tonal">
              <VIcon size="40" color="primary" :icon="benefit.icon" />
              <h3 class="text-title-large font-weight-bold mt-4 mb-2">
                {{ benefit.title }}
              </h3>
              <p class="text-body-medium text-medium-emphasis">
                {{ benefit.body }}
              </p>
            </VCard>
          </VCol>
        </VRow>
      </VContainer>
    </VSheet>

    <!-- How it works -->
    <VContainer id="how-it-works" class="py-16">
      <h2 class="text-headline-large font-weight-bold text-center mb-2">
        How it works
      </h2>
      <p class="text-body-large text-medium-emphasis text-center mb-10">
        From sign-up to payout in three steps.
      </p>
      <VRow>
        <VCol
          v-for="(step, index) in steps"
          :key="step.title"
          cols="12"
          md="4"
          :class="{ 'text-center': mdAndDown }"
        >
          <div
            class="d-flex align-center justify-center gradient-step mb-4"
            :class="{ 'mx-auto': mdAndDown }"
          >
            <span class="text-headline-small font-weight-bold">
              {{ index + 1 }}
            </span>
          </div>
          <h3 class="text-title-large font-weight-bold mb-2">
            {{ step.title }}
          </h3>
          <p class="text-body-large text-medium-emphasis">
            {{ step.body }}
          </p>
        </VCol>
      </VRow>
    </VContainer>

    <!-- FAQ -->
    <VSheet class="py-16">
      <VContainer>
        <VRow>
          <VCol cols="12" md="10" offset-md="1">
            <h2 class="text-headline-large font-weight-bold mb-10">
              Frequently asked questions
            </h2>
            <VRow>
              <VCol
                v-for="faq in faqs"
                :key="faq.question"
                cols="12"
                md="6"
                class="mb-4"
              >
                <h3 class="text-title-large font-weight-bold mb-2">
                  {{ faq.question }}
                </h3>
                <p class="text-body-large text-medium-emphasis">
                  {{ faq.answer }}
                </p>
              </VCol>
            </VRow>
          </VCol>
        </VRow>
      </VContainer>
    </VSheet>

    <!-- Closing CTA -->
    <VContainer class="py-16">
      <VRow>
        <VCol cols="12" md="8" offset-md="2">
          <VCard
            class="pa-8 text-center gradient-cta"
            :class="{ 'pa-md-16': mdAndUp }"
          >
            <h2 class="text-headline-large font-weight-bold mb-2">
              Ready to start earning?
            </h2>
            <p class="text-body-large text-medium-emphasis mb-6">
              Join free, grab your link, and turn your audience into recurring
              income.
            </p>
            <VBtn
              color="primary"
              size="large"
              :href="signupUrl"
              target="_blank"
              rel="noopener"
            >
              <VIcon start :icon="mdiCashMultiple" />
              Become an affiliate
              <VIcon end :icon="mdiArrowRightThin" />
            </VBtn>
          </VCard>
        </VCol>
      </VRow>
    </VContainer>
  </div>
</template>

<style lang="scss" scoped>
.gradient-header {
  color: #1ad37f;
  background-image: -webkit-linear-gradient(0deg, #1ad37f 14%, #329ef4 55%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.gradient-step {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background-image: -webkit-linear-gradient(0deg, #1ad37f 14%, #329ef4 55%);
  color: #fff;
}

.gradient-cta {
  border: 1px solid rgba(26, 211, 127, 0.4);
}
</style>

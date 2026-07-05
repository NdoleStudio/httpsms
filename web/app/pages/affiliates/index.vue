<script setup lang="ts">
import {
  mdiCashMultiple,
  mdiCreation,
  mdiLinkVariant,
  mdiWalletOutline,
  mdiCalendarClock,
  mdiCursorDefaultClick,
  mdiArrowRightThin,
  mdiChevronDown,
} from '@mdi/js'

definePageMeta({
  layout: 'website',
})

useSeoMeta({
  title: 'Affiliate Program — Earn 20% on every sale | httpSMS',
  description:
    'Become a httpSMS affiliate and earn 20% commission — up to $70.00 — on every customer you refer. Free to join, powered by Lemon Squeezy.',
  ogTitle: 'Get paid to share httpSMS — earn 20% on every sale',
  ogDescription:
    'Become a httpSMS affiliate and earn 20% commission — up to $70.00 — on every customer you refer. Free to join, powered by Lemon Squeezy.',
  ogImage: 'https://httpsms.com/header.png',
  twitterCard: 'summary_large_image',
})

const { mdAndDown } = useDisplay()

const signupUrl = 'https://affiliates.lemonsqueezy.com/programs/httpsms'

const highlights = [
  { value: '20%', label: 'commission on every sale' },
  { value: '$70', label: 'max earned per referral' },
  { value: 'NET30', label: 'payout terms' },
  { value: '$10', label: 'minimum payout' },
]

const faqs = [
  {
    icon: mdiCreation,
    question: 'Why promote us?',
    answer:
      "Affiliate partnerships are one of the simplest yet most profitable ways to make money online. You send customers our way, and we'll pay you for it.",
  },
  {
    icon: mdiLinkVariant,
    question: 'How do I make money as an affiliate?',
    answer:
      'All you have to do is recommend us using your affiliate link on your website, blog, and social media. We track your clicks and transactions so you can get paid.',
  },
  {
    icon: mdiCashMultiple,
    question: 'How much can I earn?',
    answer:
      "You earn 20% commission — up to $70.00 — on every referral that results in a successful sale. There's no limit to how much you can make by promoting us.",
  },
  {
    icon: mdiWalletOutline,
    question: 'What are the payout minimums?',
    answer:
      'We require a minimum balance of $10.00 before processing an affiliate payout. We set this to avoid any fraudulent issues with our affiliate program.',
  },
  {
    icon: mdiCalendarClock,
    question: 'When do I get paid?',
    answer:
      'We payout on NET30 terms to account for refunds and chargebacks. For example, commissions generated in January would be paid out on March 15th (NET30).',
  },
  {
    icon: mdiCursorDefaultClick,
    question: 'How do I sign up?',
    answer:
      'The Lemon Squeezy affiliate hub hosts our affiliate program. Click the "Become an affiliate" button below to create a Lemon Squeezy account and join our program.',
  },
]

const scrollToFaq = () => {
  document.getElementById('faq')?.scrollIntoView({ behavior: 'smooth' })
}
</script>

<template>
  <div>
    <!-- Hero -->
    <VContainer>
      <VRow>
        <VCol cols="12" md="8" offset-md="2" class="pt-16 pb-8 text-center">
          <VChip color="pink" label class="mb-6">
            <VIcon start :icon="mdiCashMultiple" />
            Affiliate Program
          </VChip>
          <h1 class="text-display-large font-weight-bold pb-1">
            Get paid to share httpSMS
          </h1>
          <h2
            class="text-medium-emphasis text-headline-small font-weight-light mt-8 mb-8"
          >
            Earn
            <span class="text-primary font-weight-bold">20% commission</span> —
            up to <span class="text-primary font-weight-bold">$70.00</span> — on
            every customer you refer. Recommend httpSMS on your blog, videos, or
            social, and we'll track every click and pay you.
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
              @click="scrollToFaq"
            >
              How it works
              <VIcon end :icon="mdiChevronDown" />
            </VBtn>
          </div>
        </VCol>
      </VRow>
    </VContainer>

    <!-- Highlights -->
    <VSheet class="py-16">
      <VContainer>
        <VRow>
          <VCol
            v-for="highlight in highlights"
            :key="highlight.label"
            cols="6"
            md="3"
            class="text-center"
          >
            <p class="text-display-medium font-weight-bold text-primary mb-1">
              {{ highlight.value }}
            </p>
            <p class="text-title-large font-weight-light text-medium-emphasis">
              {{ highlight.label }}
            </p>
          </VCol>
        </VRow>
      </VContainer>
    </VSheet>

    <!-- FAQ -->
    <VContainer id="faq" class="py-16">
      <VRow>
        <VCol cols="12" md="8" offset-md="2">
          <h2
            class="text-md-display-large text-display-medium text-center mb-4"
          >
            Frequently Asked Questions
          </h2>
          <p class="text-center text-title-large text-medium-emphasis mb-8">
            Everything you need to know before you start earning with httpSMS.
          </p>
        </VCol>
      </VRow>
      <VRow>
        <VCol
          v-for="faq in faqs"
          :key="faq.question"
          cols="12"
          md="6"
          class="py-6"
        >
          <div class="d-flex" :class="{ 'flex-column text-center': mdAndDown }">
            <VIcon
              size="32"
              color="primary"
              class="mr-4 mb-2"
              :class="{ 'mx-auto': mdAndDown }"
              :icon="faq.icon"
            />
            <div>
              <h3 class="text-headline-small font-weight-bold mb-2">
                {{ faq.question }}
              </h3>
              <p
                class="text-title-large font-weight-light text-medium-emphasis"
              >
                {{ faq.answer }}
              </p>
            </div>
          </div>
        </VCol>
      </VRow>
    </VContainer>

    <!-- Closing CTA -->
    <VSheet class="py-16">
      <VContainer>
        <VRow>
          <VCol cols="12" md="8" offset-md="2" class="text-center">
            <h2 class="text-display-medium font-weight-bold mb-4">
              Ready to start earning?
            </h2>
            <p
              class="text-title-large font-weight-light text-medium-emphasis mb-8"
            >
              Join free, grab your link, and turn your audience into recurring
              income.
            </p>
            <VBtn
              color="primary"
              size="x-large"
              :href="signupUrl"
              target="_blank"
              rel="noopener"
            >
              <VIcon start :icon="mdiCashMultiple" />
              Become an affiliate
              <VIcon end :icon="mdiArrowRightThin" />
            </VBtn>
          </VCol>
        </VRow>
      </VContainer>
    </VSheet>
  </div>
</template>

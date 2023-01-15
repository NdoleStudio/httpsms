<template>
  <v-container class="mt-16">
    <v-row>
      <v-col cols="10" offset="1" class="mt-8">
        <h1 class="text-h3 mb-4">Privacy Policy</h1>
        <p>
          Ndole Studio built the HTTP SMS service as an open source project.
          This SERVICE is provided by Ndole Studio and is intended for use as
          is. This page is used to inform visitors regarding our policies with
          the collection, use, and disclosure of Personal Information if anyone
          decided to use our Service. If you choose to use our Service, then you
          agree to the collection and use of information in relation to this
          policy. The Personal Information that we collect is used for providing
          and improving the Service. We will not use or share your information
          with anyone except as described in this Privacy Policy. The terms used
          in this Privacy Policy have the same meanings as in our Terms and
          Conditions, which are accessible at HTTP SMS unless otherwise defined
          in this Privacy Policy.
        </p>
        <h2 class="text-h6 text--secondary">Information Collection and Use</h2>
        <p>
          For a better experience, while using our Service, we may require you
          to provide us with certain personally identifiable information,
          including but not limited to email,full name. The information that we
          request will be retained by us and used as described in this privacy
          policy.
        </p>
        <p>
          HTTP SMS does use third-party services that may collect information
          used to identify you. Link to the privacy policy of third-party
          service providers used by HTTP SMS
        </p>
        <ul class="mb-4">
          <li>
            <a
              href="https://www.google.com/policies/privacy/"
              class="text-decoration-none"
              target="_blank"
              rel="noopener noreferrer"
              >Google Play Services</a
            >
          </li>
          <li>
            <a
              href="https://firebase.google.com/policies/analytics"
              class="text-decoration-none"
              target="_blank"
              rel="noopener noreferrer"
              >Google Analytics for Firebase</a
            >
          </li>
          <li>
            <a
              href="https://sentry.io/privacy/"
              target="_blank"
              class="text-decoration-none"
              rel="noopener noreferrer"
              >Sentry</a
            >
          </li>
          <li>
            <a
              href="https://segment.com/legal/privacy/"
              target="_blank"
              class="text-decoration-none"
              rel="noopener noreferrer"
              >Segment</a
            >
          </li>
        </ul>
        <h2 class="text-h6 text--secondary">Log Data</h2>
        <p>
          We want to inform you that whenever you use our Service, in a case of
          an error in the app we collect data and information (through
          third-party products) on your phone and computer called Log Data. This
          Log Data may include information such as your device Internet Protocol
          (“IP”) address, device name, operating system version, the
          configuration of the app when utilizing our Service, the time and date
          of your use of the Service, and other statistics.
        </p>
        <h2 class="text-h6 text--secondary">Cookies</h2>
        <p>
          Cookies are files with a small amount of data that are commonly used
          as anonymous unique identifiers. These are sent to your browser from
          the websites that you visit and are stored on your device's internal
          memory. This Service does not use these “cookies” explicitly. However,
          the app may use third-party code and libraries that use “cookies” to
          collect information and improve their services. You have the option to
          either accept or refuse these cookies and know when a cookie is being
          sent to your device. If you choose to refuse our cookies, you may not
          be able to use some portions of this Service.
        </p>
        <h2 class="text-h6 text--secondary">Service Providers</h2>
        <p>
          We may employ third-party companies and individuals due to the
          following reasons:
        </p>
        <ul>
          <li>To facilitate our Service;</li>
          <li>To provide the Service on our behalf;</li>
          <li>To perform Service-related services; or</li>
          <li>To assist us in analyzing how our Service is used.</li>
        </ul>
        <p>
          We want to inform users of this Service that these third parties have
          access to their Personal Information. The reason is to perform the
          tasks assigned to them on our behalf. However, they are obligated not
          to disclose or use the information for any other purpose.
        </p>
        <h2 class="text-h6 text--secondary">Security</h2>
        <p>
          We value your trust in providing us your Personal Information, thus we
          are striving to use commercially acceptable means of protecting it.
          But remember that no method of transmission over the internet, or
          method of electronic storage is 100% secure and reliable, and we
          cannot guarantee its absolute security.
        </p>
        <h2 class="text-h6 text--secondary">Links to Other Sites</h2>
        <p>
          This Service may contain links to other sites. If you click on a
          third-party link, you will be directed to that site. Note that these
          external sites are not operated by us. Therefore, we strongly advise
          you to review the Privacy Policy of these websites. We have no control
          over and assume no responsibility for the content, privacy policies,
          or practices of any third-party sites or services.
        </p>
        <h2 class="text-h6 text--secondary">Children’s Privacy</h2>
        <p>
          These Services do not address anyone under the age of 13. We do not
          knowingly collect personally identifiable information from children
          under 13 years of age. In the case we discover that a child under 13
          has provided us with personal information, we immediately delete this
          from our servers. If you are a parent or guardian and you are aware
          that your child has provided us with personal information, please
          contact us so that we will be able to do the necessary actions.
        </p>
        <h2 class="text-h6 text--secondary">Changes to This Privacy Policy</h2>
        <p>
          We may update our Privacy Policy from time to time. Thus, you are
          advised to review this page periodically for any changes. We will
          notify you of any changes by posting the new Privacy Policy on this
          page.
        </p>
        <p>This policy is effective as of 2022-10-09</p>
        <h2 class="text-h6 text--secondary">Contact Us</h2>
        <p>
          If you have any questions or suggestions about our Privacy Policy, do
          not hesitate to contact us at
          <a
            class="text-decoration-none font-weight-bold"
            href="mailto:support@httpsms.com"
            >support@httpsms.com</a
          >.
        </p>
        <v-divider class="mx-16"></v-divider>
        <div class="text-center mt-8 mb-4">
          <back-button></back-button>
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  name: 'PrivacyIndex',
  layout: 'website',
  data() {
    return {
      to: '/',
    }
  },
  head() {
    return {
      title: 'Privacy Policy - httpSMS',
    }
  },
})
</script>

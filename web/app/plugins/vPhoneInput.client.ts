import "flag-icons/css/flag-icons.min.css";
import "v-phone-input/styles";
import {
  createVPhoneInput,
  autocompletePhoneCountryInput,
  VPhoneCountryFlagSvg,
} from "v-phone-input";

export default defineNuxtPlugin((nuxtApp) => {
  const vPhoneInput = createVPhoneInput({
    ...autocompletePhoneCountryInput,
    countryDisplayComponent: VPhoneCountryFlagSvg,
    validate: null,
  });

  nuxtApp.vueApp.use(vPhoneInput);
});

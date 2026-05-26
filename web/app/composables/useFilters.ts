import {
  formatPhoneNumber,
  phoneCountry,
  formatTimestamp,
  formatMoney,
  formatDecimal,
  formatBillingPeriod,
  humanizeTime,
} from "~/utils/filters";
import { capitalize } from "~/utils/capitalize";

export function useFilters() {
  return {
    formatPhoneNumber,
    phoneCountry,
    formatTimestamp,
    formatMoney,
    formatDecimal,
    formatBillingPeriod,
    humanizeTime,
    capitalize,
  };
}

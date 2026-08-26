import {
  formatPhoneNumber,
  phoneCountry,
  formatTimestamp,
  formatMoney,
  formatDecimal,
  formatBillingPeriod,
  formatBillingPeriodDateOrdinal,
  humanizeTime,
  humanizeTimeShort,
  startsWithLetter,
} from '../utils/filters'
import { capitalize } from '../utils/capitalize'

export function useFilters() {
  return {
    formatPhoneNumber,
    phoneCountry,
    formatTimestamp,
    formatMoney,
    formatDecimal,
    formatBillingPeriod,
    humanizeTimeShort,
    formatBillingPeriodDateOrdinal,
    humanizeTime,
    startsWithLetter,
    capitalize,
  }
}

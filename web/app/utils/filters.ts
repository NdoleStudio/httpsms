import { intervalToDuration, formatDuration } from "date-fns";
import { parsePhoneNumber, isValidPhoneNumber } from "libphonenumber-js";

export function formatPhoneNumber(value: string): string {
  if (!isValidPhoneNumber(value)) {
    return value;
  }
  const phoneNumber = parsePhoneNumber(value);
  if (phoneNumber) {
    return phoneNumber.formatInternational();
  }
  return value;
}

export function phoneCountry(value: string): string {
  const phoneNumber = parsePhoneNumber(value);
  if (phoneNumber && phoneNumber.country) {
    const regionNames = new Intl.DisplayNames(undefined, { type: "region" });
    return regionNames.of(phoneNumber.country) ?? "Earth";
  }
  return "Earth";
}

export function formatTimestamp(value: string): string {
  return new Date(value).toLocaleString();
}

export function formatMoney(value: string | number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(typeof value === "string" ? parseInt(value) : value);
}

export function formatDecimal(value: string | number): string {
  return new Intl.NumberFormat("en-US", {
    style: "decimal",
  }).format(typeof value === "string" ? parseInt(value) : value);
}

export function formatBillingPeriod(value: string): string {
  return new Date(value).toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
  });
}

export function humanizeTime(value: string): string {
  const durations = intervalToDuration({
    start: new Date(),
    end: new Date(value),
  });
  return formatDuration(durations);
}

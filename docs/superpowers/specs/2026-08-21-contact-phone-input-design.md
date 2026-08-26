# Contact Phone Input Design

## Goal

Use the existing `v-phone-input` component for phone numbers in the contact
add/edit dialog.

## Design

Represent each form phone row with a phone number value and its own country.
Bind those fields to `v-phone-input` using `v-model` and `v-model:country`,
following the messages page configuration.

Existing E.164 phone numbers derive their country with `libphonenumber-js`.
Numbers without a detectable country and newly added rows default to `US`.
Removing a row removes both its number and country because they are stored in
one object.

## Behavior

- Keep support for multiple phone numbers.
- Preserve current required-phone validation and server error display.
- Submit only trimmed, non-empty phone number values to the API.
- Keep add, edit, and remove behavior unchanged apart from the country-aware
  input.

## Validation

Run ESLint, Stylelint, and Prettier against the changed contact page.

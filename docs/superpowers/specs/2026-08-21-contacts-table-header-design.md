# Contacts Table Header Styling Design

## Goal

Make every header in the contacts data table uppercase and grey using Vuetify's
standard text utility classes.

## Design

Add the `headers` slot to the existing `VDataTableServer` in
`web/app/pages/contacts/index.vue`. Render each header title with
`text-uppercase text-medium-emphasis` while retaining Vuetify's existing header
layout and per-column alignment.

This approach keeps presentation in the template, avoids brittle selectors
against Vuetify internals, and leaves the header definitions and table data flow
unchanged.

## Scope

- Apply the styling to Name, Phone Numbers, Emails, Created, Updated, and
  Actions.
- Preserve the existing data, pagination, loading, and item slot behavior.
- Do not introduce component-level CSS or change the displayed header labels.

## Validation

Run the existing web lint command and confirm the contacts page template passes
Vue and TypeScript validation.

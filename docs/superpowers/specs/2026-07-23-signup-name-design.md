# Signup Name Design

## Goal

Collect a required name during Firebase email/password signup and persist it as
the Firebase user's display name.

## User Interface

- Show a `Name` field only when the email form is in signup mode.
- Keep the existing email and password fields unchanged.
- Clear field-level and general errors whenever the user switches between sign
  in and sign up.

## Signup Flow

1. Validate name, email, and password.
2. Create the Firebase email/password user.
3. Set the new user's Firebase profile `displayName` to the trimmed name.
4. Update the auth store and continue through the existing success redirect.

## Error Handling

- An empty name produces an inline error on the `Name` field.
- Firebase account creation and profile update errors use the existing Firebase
  error handling path.
- Switching form modes removes stale errors before rendering the other mode.

## Validation

- Add or update focused component tests if the component has an existing test
  harness.
- Run the web lint command and the smallest relevant web test command.

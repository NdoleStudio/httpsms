# Message Thread Archive UI Design

## Goal

Improve message-thread selection styling and keep archive actions within the
current thread-list filter.

## Active Thread Styling

Add `color="primary"` to every thread `v-list-item` in
`web/app/components/MessageThread.vue`. Vuetify will apply the primary color
when the route-backed item is active without changing inactive items.

## Archive and Unarchive Behavior

`useThreadsStore.updateThread` will continue to send the existing
`PUT /v1/message-threads/:id` request. After a successful response, it will:

1. Preserve the current `archivedThreads` filter.
2. Remove the moved thread from the currently displayed thread collection.
3. Clear the selected thread ID.
4. Show a success notification containing `Archived` or `Unarchived`.

The thread page will route to `/threads` after the store update succeeds.
Archiving therefore returns to the unarchived list, while unarchiving returns
to the archived list. Neither action switches the list filter.

## Error Handling

API errors will continue to propagate from `apiFetch`. Local thread state,
notifications, and navigation will only change after a successful response.

## Validation

The web package has no configured automated tests. Validate the changed files
with the existing lint commands and run the existing static generation command
to confirm the production frontend builds successfully.

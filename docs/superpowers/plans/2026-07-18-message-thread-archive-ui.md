# Message Thread Archive UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Highlight the active message thread with the primary color and make archive actions return to the current filtered thread list with a success notification.

**Architecture:** Keep archive state transitions in the Pinia threads store: perform the API request, remove the moved thread from the visible collection, clear the selection, and notify. Keep route navigation in the thread page, and use Vuetify's built-in active-item color handling for the list styling.

**Tech Stack:** Nuxt 4, Vue 3, Pinia 3, Vuetify 4, TypeScript, pnpm

## Global Constraints

- Preserve the current `archivedThreads` filter after archive and unarchive actions.
- Show exactly `Archived` or `Unarchived` as a success notification after a successful update.
- Do not change local thread state or navigate when the API request fails.
- Use existing dependencies and validation commands only; the web package has no configured automated test runner.
- Include the repository's `Co-authored-by` and `Copilot-Session` trailers in every commit.

---

### Task 1: Active Thread Primary Color

**Files:**
- Modify: `web/app/components/MessageThread.vue:94-100`

**Interfaces:**
- Consumes: `threadsStore.threadId` and each thread's route-backed `v-list-item`
- Produces: Vuetify active-item styling through `color="primary"`

- [ ] **Step 1: Run the source assertion to verify the active color is missing**

Run from `web/`:

```powershell
node -e "const fs=require('fs');const s=fs.readFileSync('app/components/MessageThread.vue','utf8');if(!/<v-list-item[\s\S]*?color=\"primary\"[\s\S]*?:active=\"threadsStore\.threadId === thread\.id\"/.test(s))throw new Error('active thread primary color missing')"
```

Expected: FAIL with `Error: active thread primary color missing`.

- [ ] **Step 2: Add the primary color to every rendered thread item**

Change the list item to:

```vue
<v-list-item
  v-for="thread in threadsStore.threads"
  :key="thread.id"
  color="primary"
  :to="{ name: 'threads-id', params: { id: thread.id } }"
  :active="threadsStore.threadId === thread.id"
>
```

- [ ] **Step 3: Re-run the source assertion**

Run from `web/`:

```powershell
node -e "const fs=require('fs');const s=fs.readFileSync('app/components/MessageThread.vue','utf8');if(!/<v-list-item[\s\S]*?color=\"primary\"[\s\S]*?:active=\"threadsStore\.threadId === thread\.id\"/.test(s))throw new Error('active thread primary color missing')"
```

Expected: exit code 0.

- [ ] **Step 4: Lint the component**

Run from `web/`:

```powershell
pnpm exec eslint app/components/MessageThread.vue
pnpm exec stylelint app/components/MessageThread.vue
pnpm exec prettier --check app/components/MessageThread.vue
```

Expected: all commands exit with code 0.

- [ ] **Step 5: Commit**

```powershell
git add web/app/components/MessageThread.vue
git commit -m "fix(web): highlight active thread" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`nCopilot-Session: bf7ad4f0-3e1b-4f2d-9587-fe5b0dfae6dc"
```

### Task 2: Archive Without Switching Filters

**Files:**
- Modify: `web/app/stores/threads.ts:78-91`
- Modify: `web/app/pages/threads/[id]/index.vue:146-166`

**Interfaces:**
- Consumes: `updateThread({ threadId: string, isArchived: boolean }): Promise<void>`
- Produces: successful archive updates that remove the moved thread, clear `threadId`, notify, and allow the page to navigate to `/threads`

- [ ] **Step 1: Run source assertions to verify the required behavior is missing**

Run from `web/`:

```powershell
node -e "const fs=require('fs');const s=fs.readFileSync('app/stores/threads.ts','utf8');for(const x of [\"threads.value = threads.value.filter\",'threadId.value = null',\"payload.isArchived ? 'Archived' : 'Unarchived'\"])if(!s.includes(x))throw new Error('archive state transition missing: '+x)"
node -e "const fs=require('fs');const s=fs.readFileSync('app/pages/threads/[id]/index.vue','utf8');const matches=s.match(/await router\.push\('\/threads'\)/g)||[];if(matches.length<3)throw new Error('archive navigation missing')"
```

Expected: both commands FAIL because the store still switches `archivedThreads` and the archive actions do not navigate.

- [ ] **Step 2: Replace the store's archive-filter switch and reload**

Replace the statements after the successful `apiFetch` call in `updateThread` with:

```ts
threads.value = threads.value.filter(
  (thread) => thread.id !== payload.threadId,
)
threadId.value = null
notificationsStore.addNotification({
  message: payload.isArchived ? 'Archived' : 'Unarchived',
  type: 'success',
})
```

The complete function must remain:

```ts
async function updateThread(payload: {
  threadId: string
  isArchived: boolean
}) {
  await apiFetch(`/v1/message-threads/${payload.threadId}`, {
    method: 'PUT',
    body: { is_archived: payload.isArchived },
  })
  threads.value = threads.value.filter(
    (thread) => thread.id !== payload.threadId,
  )
  threadId.value = null
  notificationsStore.addNotification({
    message: payload.isArchived ? 'Archived' : 'Unarchived',
    type: 'success',
  })
}
```

- [ ] **Step 3: Route to the current filtered list after archive**

Change `archiveThread` to:

```ts
async function archiveThread() {
  await threadsStore.updateThread({
    threadId: threadsStore.currentThread!.id,
    isArchived: true,
  })
  await router.push('/threads')
}
```

- [ ] **Step 4: Route to the current filtered list after unarchive**

Change `unArchiveThread` to:

```ts
async function unArchiveThread() {
  await threadsStore.updateThread({
    threadId: threadsStore.currentThread!.id,
    isArchived: false,
  })
  await router.push('/threads')
}
```

- [ ] **Step 5: Re-run the source assertions**

Run from `web/`:

```powershell
node -e "const fs=require('fs');const s=fs.readFileSync('app/stores/threads.ts','utf8');for(const x of [\"threads.value = threads.value.filter\",'threadId.value = null',\"payload.isArchived ? 'Archived' : 'Unarchived'\"])if(!s.includes(x))throw new Error('archive state transition missing: '+x);if(s.includes('archivedThreads.value = payload.isArchived'))throw new Error('archive filter still switches')"
node -e "const fs=require('fs');const s=fs.readFileSync('app/pages/threads/[id]/index.vue','utf8');const matches=s.match(/await router\.push\('\/threads'\)/g)||[];if(matches.length<3)throw new Error('archive navigation missing')"
```

Expected: both commands exit with code 0.

- [ ] **Step 6: Lint the changed behavior**

Run from `web/`:

```powershell
pnpm exec eslint app/stores/threads.ts "app/pages/threads/[id]/index.vue"
pnpm exec stylelint "app/pages/threads/[id]/index.vue"
pnpm exec prettier --check app/stores/threads.ts "app/pages/threads/[id]/index.vue"
```

Expected: all commands exit with code 0.

- [ ] **Step 7: Commit**

```powershell
git add web/app/stores/threads.ts "web/app/pages/threads/[id]/index.vue"
git commit -m "fix(web): preserve thread archive filter" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`nCopilot-Session: bf7ad4f0-3e1b-4f2d-9587-fe5b0dfae6dc"
```

### Task 3: Production Validation

**Files:**
- Verify: `web/app/components/MessageThread.vue`
- Verify: `web/app/stores/threads.ts`
- Verify: `web/app/pages/threads/[id]/index.vue`

**Interfaces:**
- Consumes: completed UI and archive behavior from Tasks 1 and 2
- Produces: lint-clean, production-generated frontend output

- [ ] **Step 1: Run all targeted lint checks together**

Run from `web/`:

```powershell
pnpm exec eslint app/components/MessageThread.vue app/stores/threads.ts "app/pages/threads/[id]/index.vue"
pnpm exec stylelint app/components/MessageThread.vue "app/pages/threads/[id]/index.vue"
pnpm exec prettier --check app/components/MessageThread.vue app/stores/threads.ts "app/pages/threads/[id]/index.vue"
```

Expected: all commands exit with code 0.

- [ ] **Step 2: Generate the production static site**

Run from `web/`:

```powershell
pnpm generate
```

Expected: Nuxt generation completes successfully and writes the static output.

- [ ] **Step 3: Check the final diff**

Run from the repository root:

```powershell
git diff main...HEAD --check
git status --short
```

Expected: no whitespace errors and a clean working tree.

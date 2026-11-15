import { defineStore } from 'pinia'
import { computed, onScopeDispose, ref } from 'vue'
import {
  BLACK_FRIDAY_END,
  getBlackFridayCountdown,
} from '../utils/black-friday'

export const useBlackFridayStore = defineStore('blackFriday', () => {
  const now = ref(Date.now())
  const active = computed(() => now.value < BLACK_FRIDAY_END)
  const countdown = computed(() => getBlackFridayCountdown(now.value))
  let timer: ReturnType<typeof setInterval> | undefined

  if (active.value) {
    timer = setInterval(() => {
      now.value = Date.now()
      if (!active.value) clearInterval(timer)
    }, 1000)
  }

  onScopeDispose(() => clearInterval(timer))

  return { now, active, countdown }
})

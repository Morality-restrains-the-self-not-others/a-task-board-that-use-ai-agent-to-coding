<template>
  <div
    data-alias="view-acknowledgments-page"
    data-testid="view-acknowledgments-page"
    class="min-h-screen bg-gray-50 font-sans pt-24 pb-16 px-4 sm:px-6 lg:px-8"
  >
    <div class="max-w-3xl mx-auto">
      <h1 class="text-3xl font-bold text-gray-900 mb-2">致谢</h1>
      <p class="text-gray-600 mb-8">
        本平台的能力建设受益于开源社区。特此致谢下列项目与贡献者。
      </p>

      <ul class="space-y-6" data-testid="acknowledgments-list">
        <li
          v-for="entry in entries"
          :key="entry.id"
          class="bg-white rounded-lg shadow-sm border border-gray-200 p-6"
          :data-testid="`ack-entry-${entry.id}`"
        >
          <h2 class="text-lg font-semibold text-gray-800 mb-2">{{ entry.name }}</h2>
          <p class="text-gray-700 mb-4">{{ entry.summary }}</p>
          <ul class="list-disc list-inside space-y-1 text-gray-600 text-sm mb-4">
            <li v-for="(item, idx) in entry.highlights" :key="`${entry.id}-h-${idx}`">
              {{ item }}
            </li>
          </ul>
          <a
            :href="entry.repoUrl"
            :data-testid="`ack-repo-link-${entry.id}`"
            class="text-primary font-medium hover:underline"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ entry.repoUrl }}
          </a>
        </li>
      </ul>

      <p class="mt-10 text-sm text-gray-500">
        <router-link
          :to="{ name: 'home' }"
          class="text-primary hover:underline"
          data-testid="ack-back-home-link"
        >
          返回首页
        </router-link>
      </p>
    </div>
  </div>
</template>

<script setup>
import { ACKNOWLEDGMENT_ENTRIES } from '../data/acknowledgmentsContent.js'

/* @alias:view-acknowledgments-page */

const entries = ACKNOWLEDGMENT_ENTRIES
</script>

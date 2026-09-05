<template>
  <div v-if="visible" class="mt-3" data-testid="image-skills-list">
    <p class="text-xs font-medium text-gray-600">技能列表</p>
    <p v-if="pending" class="mt-1 text-xs text-gray-500">技能列表抽取中…</p>
    <p v-else-if="failed" class="mt-1 text-xs text-amber-700">技能列表抽取失败</p>
    <ul v-else class="mt-1 flex flex-wrap gap-1.5">
      <li
        v-for="skill in list.skills"
        :key="skill.name"
        class="inline-flex items-center rounded-md px-2 py-0.5 text-xs"
        :class="skill.isDefault ? 'bg-indigo-100 text-indigo-800 font-medium' : 'bg-gray-100 text-gray-700'"
        :data-testid="skill.isDefault ? 'image-skill-default' : 'image-skill'"
      >
        /{{ skill.name }}
        <span v-if="skill.isDefault" class="ml-1 text-[10px] uppercase tracking-wide">默认</span>
      </li>
    </ul>
    <p
      v-if="list.skills.length && list.skills[0].description"
      class="mt-1 text-xs text-gray-500 line-clamp-2"
    >
      {{ list.skills[0].description }}
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { imageSkillsFromImage, shouldShowImageSkills } from '../utils/imageSkills.js'

const props = defineProps({
  image: { type: Object, default: null },
})

const list = computed(() => imageSkillsFromImage(props.image))
const visible = computed(() => shouldShowImageSkills(props.image))
const status = computed(() => String(props.image?.image_skills_extract_status || '').trim())
const pending = computed(() => status.value === 'pending')
const failed = computed(() => status.value === 'failed' || status.value.startsWith('auth_failed'))
</script>

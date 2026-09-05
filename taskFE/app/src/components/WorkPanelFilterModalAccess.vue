<template>
  <div data-alias="filter-modal-access">
    <div class="flex items-center justify-between mb-2">
      <span class="block text-sm font-medium text-gray-700">人 / 小组</span>
      <button
        v-if="accessFilter"
        type="button"
        data-alias="filter-modal-clear-access"
        class="text-xs text-primary"
        @click="$emit('clear')"
      >清除</button>
    </div>
    <div class="flex border border-gray-200 rounded-md overflow-hidden mb-2">
      <button
        type="button"
        data-alias="filter-modal-tab-person"
        class="flex-1 px-2 py-1.5 text-sm"
        :class="tab === 'person' ? 'bg-primary/10 text-primary font-medium' : 'text-gray-600'"
        @click="$emit('update:tab', 'person')"
      >人</button>
      <button
        type="button"
        data-alias="filter-modal-tab-group"
        class="flex-1 px-2 py-1.5 text-sm"
        :class="tab === 'group' ? 'bg-primary/10 text-primary font-medium' : 'text-gray-600'"
        @click="$emit('update:tab', 'group')"
      >小组</button>
    </div>
    <div class="max-h-40 overflow-y-auto border border-gray-100 rounded-md">
      <p v-if="loading" class="px-3 py-2 text-sm text-gray-500">加载中…</p>
      <template v-else-if="tab === 'person'">
        <p v-if="!people.length" class="px-3 py-2 text-sm text-gray-500">暂无可访问成员</p>
        <button
          v-for="p in people"
          :key="`m-p-${p.id}`"
          type="button"
          data-alias="filter-modal-person-option"
          class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
          :class="isSelected('person', p.id) ? 'bg-primary/5 text-primary font-medium' : ''"
          @click="$emit('select', { kind: 'person', id: p.id, label: p.label })"
        >{{ p.label }}</button>
      </template>
      <template v-else>
        <p v-if="!groups.length" class="px-3 py-2 text-sm text-gray-500">暂无可访问小组</p>
        <button
          v-for="g in groups"
          :key="`m-g-${g.id}`"
          type="button"
          data-alias="filter-modal-group-option"
          class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
          :class="isSelected('group', g.id) ? 'bg-primary/5 text-primary font-medium' : ''"
          @click="$emit('select', { kind: 'group', id: g.id, label: g.label })"
        >{{ g.label }}</button>
      </template>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  accessFilter: { type: Object, default: null },
  people: { type: Array, default: () => [] },
  groups: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  tab: { type: String, default: 'person' },
})
defineEmits(['clear', 'select', 'update:tab'])

function isSelected(kind, id) {
  const f = props.accessFilter
  return Boolean(f && f.kind === kind && String(f.id) === String(id))
}
</script>

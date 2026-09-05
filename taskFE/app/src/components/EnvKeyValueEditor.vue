<template>
  <div class="space-y-2">
    <label class="block text-sm font-medium text-text-light mb-1">额外环境变量（键值对）</label>
    <div v-for="(row, idx) in items" :key="idx" class="flex space-x-2 mb-2">
      <input
        v-model.trim="row.key"
        type="text"
        placeholder="变量名"
        class="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary"
        @input="emitChange"
      />
      <input
        v-model.trim="row.value"
        type="text"
        placeholder="变量值"
        class="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary"
        @input="emitChange"
      />
      <button
        type="button"
        class="px-2 py-1 text-red-500 hover:bg-red-50 rounded"
        @click="removeRow(idx)"
      >✕</button>
    </div>
    <button
      type="button"
      class="px-3 py-1 text-sm border border-gray-300 rounded-md text-gray-600 hover:bg-gray-50"
      @click="addRow"
    >+ 添加变量</button>
  </div>
</template>

<script>
export default {
  name: 'EnvKeyValueEditor',
  props: {
    modelValue: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  data() {
    return {
      items: (this.modelValue || []).map(e => ({ key: e.key || '', value: e.value || '' })),
    }
  },
  watch: {
    modelValue(v) {
      const incoming = (v || []).map((e) => ({ key: e.key || '', value: e.value || '' }))
      const incomingCommitted = this.committedPayload(incoming)
      const localCommitted = this.committedPayload(this.items)
      if (this.payloadsEqual(localCommitted, incomingCommitted)) {
        return
      }
      this.items = incoming
    },
  },
  methods: {
    committedPayload(rows) {
      return (rows || [])
        .filter((r) => (r.key || '').trim() !== '')
        .map((r) => ({ key: r.key.trim(), value: r.value }))
    },
    payloadsEqual(a, b) {
      return JSON.stringify(a) === JSON.stringify(b)
    },
    addRow() {
      // 空草稿不立刻 emit，避免被 filter 后 watch 回写冲掉新行
      this.items.push({ key: '', value: '' })
    },
    removeRow(idx) {
      this.items.splice(idx, 1)
      this.emitChange()
    },
    emitChange() {
      this.$emit('update:modelValue', this.committedPayload(this.items))
    },
  },
}
</script>

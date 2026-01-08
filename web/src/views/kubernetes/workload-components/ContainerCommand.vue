<template>
  <div class="command-form-wrapper">
    <!-- 工作目录 -->
    <div class="form-row-group">
      <div class="form-row-item full-width">
        <label class="form-label">工作目录</label>
        <el-input
          v-model="localContainer.workingDir"
          placeholder="容器的工作目录，如: /app"
          clearable
          @input="update"
        />
      </div>
    </div>

    <!-- 运行命令 -->
    <div class="form-section-block">
      <label class="form-label">运行命令 (Command)</label>
      <div class="command-list">
        <div v-for="(cmd, index) in localContainer.command" :key="'cmd-'+index" class="command-item">
          <el-input v-model="localContainer.command[index]" placeholder="命令参数" @input="update">
            <template #prepend>{{ index + 1 }}</template>
          </el-input>
          <el-button type="danger" link @click="removeCommand(index)" :icon="Delete" />
        </div>
      </div>
      <el-button class="add-btn" type="primary" link @click="addCommand" :icon="Plus">
        添加命令
      </el-button>
    </div>

    <!-- 启动参数 -->
    <div class="form-section-block">
      <label class="form-label">启动参数 (Args)</label>
      <div class="command-list">
        <div v-for="(arg, index) in localContainer.args" :key="'arg-'+index" class="command-item">
          <el-input v-model="localContainer.args[index]" placeholder="参数值" @input="update">
            <template #prepend>{{ index + 1 }}</template>
          </el-input>
          <el-button type="danger" link @click="removeArg(index)" :icon="Delete" />
        </div>
      </div>
      <el-button class="add-btn" type="primary" link @click="addArg" :icon="Plus">
        添加参数
      </el-button>
    </div>

    <!-- 交互选项 -->
    <div class="form-section-block">
      <label class="form-label">交互选项</label>
      <div class="checkbox-group">
        <el-checkbox v-model="localContainer.stdin" @change="update">保持标准输入开启 (stdin)</el-checkbox>
        <el-checkbox v-model="localContainer.tty" @change="update">分配终端 (TTY)</el-checkbox>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'

interface Container {
  workingDir?: string
  command: string[]
  args: string[]
  stdin?: boolean
  tty?: boolean
}

const props = defineProps<{
  container: Container
}>()

const emit = defineEmits<{
  update: [container: Container]
}>()

const localContainer = reactive<Container>({
  workingDir: '',
  command: [],
  args: [],
  stdin: false,
  tty: false
})

watch(() => props.container, (newVal) => {
  localContainer.workingDir = newVal.workingDir || ''
  localContainer.command = newVal.command || []
  localContainer.args = newVal.args || []
  localContainer.stdin = newVal.stdin || false
  localContainer.tty = newVal.tty || false
}, { deep: true, immediate: true })

const update = () => {
  emit('update', { ...localContainer })
}

const addCommand = () => {
  if (!localContainer.command) localContainer.command = []
  localContainer.command.push('')
  update()
}

const removeCommand = (index: number) => {
  localContainer.command.splice(index, 1)
  update()
}

const addArg = () => {
  if (!localContainer.args) localContainer.args = []
  localContainer.args.push('')
  update()
}

const removeArg = (index: number) => {
  localContainer.args.splice(index, 1)
  update()
}
</script>

<style scoped>
.command-form-wrapper {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-row-group {
  display: flex;
  gap: 12px;
}

.form-row-item {
  flex: 1;
}

.form-row-item.full-width {
  flex: 100%;
}

.form-label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.form-section-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.command-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.command-item {
  display: flex;
  gap: 8px;
}

.add-btn {
  margin-top: 4px;
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>

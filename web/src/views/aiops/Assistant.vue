<template>
  <div class="assistant-page">
    <div class="assistant-hero">
      <div>
        <div class="eyebrow">AIOps Assistant</div>
        <h2>AI助手</h2>
        <p>面向运维与开发的智能问答入口，支持排障思路、命令建议、日志解释和平台能力咨询。</p>
      </div>
      <div class="hero-actions">
        <ProviderSelect v-model="providerId" placeholder="选择模型" />
        <el-button class="black-button" @click="newSession">
          <el-icon><Plus /></el-icon>
          新建会话
        </el-button>
      </div>
    </div>

    <div class="assistant-layout">
      <aside class="side-panel">
        <div class="session-sidebar-head">
          <div>
            <div class="panel-title">会话</div>
            <p>保留最近的 AI 助手问答</p>
          </div>
          <el-button class="side-new-button" @click="newSession">
            <el-icon><Plus /></el-icon>
          </el-button>
        </div>

        <el-scrollbar class="session-list-wrap">
          <div
            v-if="!sessionLoading && sessions.length === 0"
            class="session-empty"
          >
            <el-icon><MessageBox /></el-icon>
            <span>暂无会话，点击新建会话开始提问</span>
          </div>
          <div
            v-for="item in sessions"
            :key="item.id"
            class="session-item"
            :class="{ active: sessionId === item.id }"
            @click="openSession(item)"
          >
            <div class="session-item-main">
              <div class="session-title">{{ item.title || '未命名会话' }}</div>
              <div class="session-meta">
                <span>{{ formatSessionTime(item.updatedAt || item.createdAt) }}</span>
                <span>{{ item.messageCount || 0 }} 条消息</span>
              </div>
            </div>
            <el-button
              class="session-delete"
              text
              :loading="deletingSessionId === item.id"
              @click.stop="handleDeleteSession(item)"
            >
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </el-scrollbar>
      </aside>

      <main class="chat-panel">
        <div ref="messageWrapRef" class="message-wrap" @scroll="handleMessageScroll">
          <div v-if="messages.length === 0" class="empty-state">
            <div class="empty-icon">
              <el-icon><MessageBox /></el-icon>
            </div>
            <h3>问一个运维问题吧</h3>
            <p>比如“帮我分析 Pod CrashLoopBackOff 的排查流程”或“生成查看磁盘占用的安全命令”。</p>
          </div>

          <div v-for="msg in messages" :key="msg.id" class="message-row" :class="msg.role">
            <div class="avatar">
              <el-icon v-if="msg.role === 'assistant'"><DataAnalysis /></el-icon>
              <el-icon v-else><User /></el-icon>
            </div>
            <div class="bubble">
              <div class="bubble-meta">{{ msg.role === 'assistant' ? 'OpsHub AI' : '我' }}</div>
              <div v-if="msg.role === 'assistant' && msg.thinkingSteps?.length" class="thinking-card">
                <div class="thinking-title">分析过程</div>
                <div v-for="step in msg.thinkingSteps" :key="step" class="thinking-step">
                  <span></span>
                  {{ step }}
                </div>
              </div>
              <template v-if="msg.role === 'assistant'">
                <div v-if="msg.streaming && !msg.content" class="streaming-placeholder">
                  <span></span>
                  正在生成回答...
                </div>
                <MarkdownView v-if="msg.content" :content="msg.content" />
                <div v-if="msg.streaming && msg.content" class="streaming-inline">
                  <span></span>
                  正在持续生成中，请稍候...
                </div>
                <div v-if="msg.truncated || msg.incomplete" class="truncate-tip">
                  <span>{{ msg.incompleteReason || '回答被模型最大输出长度截断了，可以继续生成后面的内容。' }}</span>
                  <el-button size="small" class="continue-button" :disabled="sending" @click="continueGeneration">继续生成</el-button>
                </div>
                <div v-if="msg.content" class="answer-actions">
                  <el-button size="small" class="answer-action-button" @click="copyAnswerMarkdown(msg)">
                    <el-icon><CopyDocument /></el-icon>
                    复制 Markdown
                  </el-button>
                  <el-button size="small" class="answer-action-button primary" @click="downloadAnswerMarkdown(msg)">
                    <el-icon><Download /></el-icon>
                    下载 Markdown
                  </el-button>
                </div>
              </template>
              <pre v-else>{{ msg.content }}</pre>
            </div>
          </div>
        </div>

        <div class="input-area">
          <el-input
            v-model="input"
            type="textarea"
            :rows="4"
            resize="none"
            placeholder="输入你的问题，例如：帮我生成一套排查 Kubernetes 服务 502 的步骤"
            @keydown.meta.enter.prevent="handleSend()"
            @keydown.ctrl.enter.prevent="handleSend()"
          />
          <div class="input-footer">
            <span>Ctrl/Command + Enter 发送</span>
            <el-button v-if="sending" class="stop-button" @click="stopGeneration">
              停止生成
            </el-button>
            <el-button v-else class="black-button" :disabled="!input.trim()" @click="handleSend()">
              <el-icon><Promotion /></el-icon>
              发送
            </el-button>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { CopyDocument, DataAnalysis, Delete, Download, MessageBox, Plus, Promotion, User } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { chatWithAI, chatWithAIStream, deleteAISession, getAISessionDetail, getAISessions, stopAIChat, type AIChatPayload } from '@/api/aiops'
import MarkdownView from './components/MarkdownView.vue'
import ProviderSelect from './components/ProviderSelect.vue'

interface LocalMessage {
  id: number
  role: 'user' | 'assistant'
  content: string
  status?: string
  contextRef?: string
  thinkingSteps?: string[]
  streaming?: boolean
  truncated?: boolean
  incomplete?: boolean
  incompleteReason?: string
  finishReason?: string
  error?: string
}

interface ChatSession {
  id: number
  title: string
  summary?: string
  type?: string
  messageCount?: number
  createdAt?: string
  updatedAt?: string
}

interface SendOptions {
  content?: string
  isContinue?: boolean
  continueFromMessageId?: number
  originalQuestion?: string
  previousAnswer?: string
}

const input = ref('')
const sending = ref(false)
const sessionLoading = ref(false)
const deletingSessionId = ref<number>()
const sessionId = ref<number | undefined>()
const providerId = ref<number | undefined>()
const messages = ref<LocalMessage[]>([])
const sessions = ref<ChatSession[]>([])
const messageWrapRef = ref<HTMLElement>()
const activeController = ref<AbortController>()
const generatingPollTimer = ref<number>()
const syncingSession = ref(false)
const shouldAutoScroll = ref(true)
const programmaticScrolling = ref(false)
const completedBySessionRefresh = ref(false)
const streamDraftKey = 'opshub-aiops-assistant-stream-draft-v1'
const bottomFollowThreshold = 140

const normalizePage = (res: any) => {
  return {
    data: res?.data || [],
    total: res?.total || 0,
    page: res?.page || 1,
    pageSize: res?.page_size || res?.pageSize || 20
  }
}

const formatSessionTime = (value?: string) => {
  if (!value) return '刚刚'
  const time = new Date(value).getTime()
  if (Number.isNaN(time)) return value
  const diff = Date.now() - time
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  if (diff < minute) return '刚刚'
  if (diff < hour) return `${Math.max(1, Math.floor(diff / minute))} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 7 * day) return `${Math.floor(diff / day)} 天前`
  return new Date(value).toLocaleDateString()
}

const loadSessions = async () => {
  sessionLoading.value = true
  try {
    const res = normalizePage(await getAISessions({ page: 1, pageSize: 50, type: 'chat' }))
    sessions.value = res.data
  } finally {
    sessionLoading.value = false
  }
}

const isNearMessageBottom = () => {
  const el = messageWrapRef.value
  if (!el) return true
  return el.scrollHeight - el.scrollTop - el.clientHeight <= bottomFollowThreshold
}

const handleMessageScroll = () => {
  if (programmaticScrolling.value) return
  shouldAutoScroll.value = isNearMessageBottom()
}

const scrollToBottom = async (options: { force?: boolean } = {}) => {
  await nextTick()
  const el = messageWrapRef.value
  if (!el) return
  if (options.force) {
    shouldAutoScroll.value = true
  }
  if (!options.force && !shouldAutoScroll.value) return
  programmaticScrolling.value = true
  el.scrollTop = el.scrollHeight
  window.requestAnimationFrame(() => {
    programmaticScrolling.value = false
    shouldAutoScroll.value = isNearMessageBottom()
  })
}

const stripThinkingMarkup = (value: string) => {
  return String(value || '')
    .replace(/<think>[\s\S]*?<\/think>/gi, '')
    .replace(/<think>[\s\S]*$/gi, '')
    .replace(/<\/think>/gi, '')
    .trimStart()
}

const saveStreamDraft = () => {
  try {
    if (!messages.value.length) return
    localStorage.setItem(streamDraftKey, JSON.stringify({
      sessionId: sessionId.value,
      providerId: providerId.value,
      messages: messages.value,
      updatedAt: Date.now()
    }))
  } catch {
    // 本地缓存不可用时不影响正常聊天。
  }
}

const clearStreamDraft = () => {
  try {
    localStorage.removeItem(streamDraftKey)
  } catch {
    // ignore
  }
}

const normalizeMessageStatus = (status?: string) => String(status || '').toLowerCase()
const isGeneratingStatus = (status?: string) => normalizeMessageStatus(status) === 'generating'
const isTerminalMessageStatus = (status?: string) => ['success', 'truncated', 'interrupted', 'failed', 'error'].includes(normalizeMessageStatus(status))
const isIncompleteMessageStatus = (status?: string) => ['generating', 'interrupted', 'truncated'].includes(normalizeMessageStatus(status))

const extractThinkingStepsFromContext = (contextRef?: string): string[] => {
  if (!contextRef) return []
  try {
    const context = JSON.parse(contextRef)
    if (Array.isArray(context?.thinkingSteps)) {
      return context.thinkingSteps.filter((item: unknown) => typeof item === 'string' && item.trim())
    }
    if (Array.isArray(context?.turns)) {
      const steps: string[] = []
      context.turns.forEach((turn: any) => {
        const round = turn?.round || steps.length + 1
        if (turn?.thoughtSummary) {
          steps.push(`第 ${round} 轮思考摘要：${turn.thoughtSummary}`)
        }
        if (turn?.action?.tool) {
          steps.push(`第 ${round} 轮工具调用：${turn.action.tool}`)
        }
        if (turn?.observation?.name) {
          steps.push(`第 ${round} 轮观察结果：${turn.observation.success ? '查询成功' : turn.observation.error || '查询失败'}`)
        }
      })
      return steps
    }
  } catch {
    return []
  }
  return []
}

const normalizeSessionMessage = (msg: any): LocalMessage => {
  const localMsg: LocalMessage = {
    id: msg.id,
    role: msg.role,
    content: msg.content || '',
    status: msg.status,
    contextRef: msg.contextRef,
    thinkingSteps: extractThinkingStepsFromContext(msg.contextRef),
    error: msg.error
  }
  if (localMsg.role === 'assistant' && isGeneratingStatus(localMsg.status)) {
    localMsg.streaming = true
  } else if (localMsg.role === 'assistant' && normalizeMessageStatus(localMsg.status) === 'truncated') {
    markTruncated(localMsg)
  } else if (localMsg.role === 'assistant' && isIncompleteMessageStatus(localMsg.status)) {
    markIncomplete(localMsg, localMsg.error || (localMsg.status === 'generating'
      ? '上次回答生成过程中断，已恢复服务端保存的部分内容，可以继续生成。'
      : '上次回答没有完整结束，已恢复服务端保存的部分内容，可以继续生成。'))
  } else if (localMsg.role === 'assistant') {
    localMsg.streaming = false
  }
  return localMsg
}

const hasGeneratingMessage = () => messages.value.some(msg => msg.role === 'assistant' && isGeneratingStatus(msg.status) && msg.streaming !== false)

const stopGeneratingPoll = () => {
  if (!generatingPollTimer.value) return
  window.clearInterval(generatingPollTimer.value)
  generatingPollTimer.value = undefined
}

const applySessionDetail = (detail: any) => {
  const normalizedMessages = (detail?.messages || [])
    .map(normalizeSessionMessage)
    .filter((msg: LocalMessage) => msg.role === 'user' || msg.role === 'assistant')
  if (sending.value && messages.value.length) {
    const existingById = new Map(messages.value.map(msg => [msg.id, msg]))
    messages.value = normalizedMessages.map((serverMsg: LocalMessage) => {
      const existing = existingById.get(serverMsg.id)
      if (!existing) return serverMsg
      const serverGenerating = serverMsg.role === 'assistant' && isGeneratingStatus(serverMsg.status)
      const serverTerminal = serverMsg.role === 'assistant' && isTerminalMessageStatus(serverMsg.status)
      const merged: LocalMessage = {
        ...serverMsg,
        streaming: serverTerminal ? false : (serverGenerating ? true : Boolean(serverMsg.streaming)),
        content: existing.content && existing.content.length > serverMsg.content.length ? existing.content : serverMsg.content,
        thinkingSteps: serverMsg.thinkingSteps?.length ? serverMsg.thinkingSteps : existing.thinkingSteps
      }
      Object.assign(existing, merged)
      return existing
    })
  } else {
    messages.value = normalizedMessages
  }
  if (hasGeneratingMessage()) {
    saveStreamDraft()
  } else {
    clearStreamDraft()
    stopGeneratingPoll()
    if (sending.value) {
      completedBySessionRefresh.value = true
      sending.value = false
      if (activeController.value) {
        activeController.value.abort()
        activeController.value = undefined
      }
    }
  }
}

const refreshActiveSession = async () => {
  if (!sessionId.value || syncingSession.value) return
  syncingSession.value = true
  try {
    const detail = await getAISessionDetail(sessionId.value)
    applySessionDetail(detail)
    await scrollToBottom()
    void loadSessions()
  } finally {
    syncingSession.value = false
  }
}

const startGeneratingPoll = () => {
  if (generatingPollTimer.value) return
  generatingPollTimer.value = window.setInterval(() => {
    if (!sessionId.value || !hasGeneratingMessage()) {
      stopGeneratingPoll()
      return
    }
    void refreshActiveSession()
  }, 1200)
}

const newSession = () => {
  stopGeneration()
  stopGeneratingPoll()
  clearStreamDraft()
  shouldAutoScroll.value = true
  sessionId.value = undefined
  messages.value = []
}

const openSession = async (item: ChatSession) => {
  stopGeneration()
  if (sessionId.value === item.id) return
  stopGeneratingPoll()
  clearStreamDraft()
  sessionId.value = item.id
  const detail = await getAISessionDetail(item.id)
  applySessionDetail(detail)
  if (hasGeneratingMessage()) {
    startGeneratingPoll()
  }
  await scrollToBottom({ force: true })
}

const isLengthFinish = (reason?: string) => String(reason || '').toLowerCase() === 'length'

const cleanAnswerForContinue = (value: string) => {
  let text = String(value || '')
  let removedRuntimeNotice = false
  const markers = [
    '> 本次流式生成中断',
    '> 本次回答已到达模型最大输出长度',
    '生成被中断，当前内容已作为本地草稿保留',
    '回答被模型最大输出长度截断'
  ]
  markers.forEach((marker) => {
    const index = text.indexOf(marker)
    if (index >= 0) {
      text = text.slice(0, index)
      removedRuntimeNotice = true
    }
  })
  text = text.trimEnd()
  if (removedRuntimeNotice && /```[ \t]*$/.test(text)) {
    const withoutFence = text.replace(/\n?```[ \t]*$/, '').trimEnd()
    const fenceCount = (withoutFence.match(/```/g) || []).length
    if (fenceCount % 2 === 1) {
      text = withoutFence
    }
  }
  return text.trim()
}

const markTruncated = (msg: LocalMessage) => {
  msg.truncated = true
  msg.incomplete = false
  msg.status = 'truncated'
  msg.incompleteReason = '回答被模型最大输出长度截断了，可以继续生成后面的内容。'
}

const markIncomplete = (msg: LocalMessage, reason: string) => {
  msg.incomplete = true
  msg.status = msg.status || 'interrupted'
  msg.incompleteReason = reason
}

const markdownFileName = (msg: LocalMessage) => {
  const stamp = new Date()
    .toISOString()
    .replace(/T/, '-')
    .replace(/\..+$/, '')
    .replace(/:/g, '')
  return `opshub-ai-answer-${stamp}-${msg.id}.md`
}

const fallbackCopyText = (content: string) => {
  const textarea = document.createElement('textarea')
  textarea.value = content
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  textarea.style.top = '-9999px'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(textarea)
  return ok
}

const copyAnswerMarkdown = async (msg: LocalMessage) => {
  const content = msg.content || ''
  if (!content.trim()) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(content)
    } else if (!fallbackCopyText(content)) {
      throw new Error('copy failed')
    }
    ElMessage.success('已复制 Markdown')
  } catch {
    ElMessage.error('复制失败，请手动选择内容复制')
  }
}

const downloadAnswerMarkdown = (msg: LocalMessage) => {
  const content = msg.content || ''
  if (!content.trim()) return
  const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = markdownFileName(msg)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

const restoreStreamDraft = () => {
  try {
    const raw = localStorage.getItem(streamDraftKey)
    if (!raw) return
    const draft = JSON.parse(raw)
    if (!draft?.messages?.length) {
      clearStreamDraft()
      return
    }
    if (Date.now() - Number(draft.updatedAt || 0) > 24 * 60 * 60 * 1000) {
      clearStreamDraft()
      return
    }
    sessionId.value = draft.sessionId
    providerId.value = draft.providerId
    messages.value = (draft.messages || [])
      .filter((msg: LocalMessage) => msg?.role === 'user' || msg?.role === 'assistant')
      .map((msg: LocalMessage) => ({
        ...msg,
        streaming: msg.role === 'assistant' && isGeneratingStatus(msg.status)
      }))
    const lastAssistant = [...messages.value].reverse().find(msg => msg.role === 'assistant')
    if (lastAssistant && isGeneratingStatus(lastAssistant.status)) {
      startGeneratingPoll()
    } else if (lastAssistant && !lastAssistant.truncated) {
      markIncomplete(lastAssistant, lastAssistant.content
        ? '页面刷新前回答还没有完成，已恢复本地草稿，可以继续生成。'
        : '页面刷新前回答还没有开始输出，已恢复提问记录，可以继续生成。')
    }
  } catch {
    clearStreamDraft()
  }
}

const stopGeneration = async () => {
  const streamingMsg = [...messages.value].reverse().find(msg => msg.role === 'assistant' && msg.streaming)
  const serverMessageId = streamingMsg?.id && streamingMsg.id < 1000000000000 ? streamingMsg.id : undefined
  if (sessionId.value && serverMessageId) {
    stopAIChat({ sessionId: sessionId.value, messageId: serverMessageId }).catch(() => undefined)
  } else if (sessionId.value) {
    stopAIChat({ sessionId: sessionId.value }).catch(() => undefined)
  }
  if (!activeController.value && !streamingMsg) return
  if (activeController.value) {
    activeController.value.abort()
    activeController.value = undefined
  }
  sending.value = false
  if (streamingMsg) {
    streamingMsg.streaming = false
    if (!streamingMsg.content) {
      streamingMsg.content = '已停止生成。'
    } else {
      markIncomplete(streamingMsg, '已停止生成，当前内容已作为本地草稿保留，可以继续生成。')
    }
    saveStreamDraft()
  }
}

const findOriginalQuestionBefore = (messageId: number) => {
  const index = messages.value.findIndex(msg => msg.id === messageId)
  const searchEnd = index >= 0 ? index : messages.value.length
  for (let i = searchEnd - 1; i >= 0; i -= 1) {
    const msg = messages.value[i]
    if (msg?.role !== 'user') continue
    const content = msg.content.trim()
    if (!content) continue
    if (content.includes('请从上次中断的位置继续输出') || content.includes('不要重复已经输出的内容')) continue
    return content
  }
  return ''
}

const continueGeneration = () => {
  if (sending.value) return
  const assistantMsg = [...messages.value].reverse().find(msg => msg.role === 'assistant' && (msg.incomplete || msg.truncated || msg.status === 'interrupted' || msg.status === 'generating'))
  if (!assistantMsg) {
    ElMessage.warning('没有找到可继续生成的回答')
    return
  }
  const originalQuestion = findOriginalQuestionBefore(assistantMsg.id)
  void handleSend({
    content: '请从上次中断的位置继续输出，不要重复已经输出的内容。',
    isContinue: true,
    continueFromMessageId: assistantMsg.id,
    originalQuestion,
    previousAnswer: cleanAnswerForContinue(assistantMsg.content)
  })
}

const handleDeleteSession = async (item: ChatSession) => {
  try {
    await ElMessageBox.confirm(`确定删除会话「${item.title || '未命名会话'}」吗？删除后不可恢复。`, '删除会话', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger'
    })
  } catch {
    return
  }
  deletingSessionId.value = item.id
  try {
    await deleteAISession(item.id)
    ElMessage.success('会话已删除')
    if (sessionId.value === item.id) {
      clearStreamDraft()
      newSession()
    }
    await loadSessions()
  } finally {
    deletingSessionId.value = undefined
  }
}

const handleSend = async (options?: SendOptions) => {
  const content = (options?.content ?? input.value).trim()
  if (!content || sending.value) return
  if (!options?.isContinue) {
    const userMsg: LocalMessage = { id: Date.now(), role: 'user', content }
    messages.value.push(userMsg)
    saveStreamDraft()
  }
  if (!options?.content) {
    input.value = ''
  }
  await scrollToBottom({ force: true })
  sending.value = true
  completedBySessionRefresh.value = false
  const controller = new AbortController()
  activeController.value = controller
  const payload: AIChatPayload = {
    sessionId: sessionId.value,
    message: content,
    providerId: providerId.value,
    continue: options?.isContinue,
    continueFromMessageId: options?.continueFromMessageId,
    originalQuestion: options?.originalQuestion,
    previousAnswer: options?.previousAnswer
  }
  try {
    let hasStreamDelta = false
    const assistantMsg: LocalMessage = {
      id: Date.now() + 1,
      role: 'assistant',
      content: '',
      thinkingSteps: [],
      streaming: true
    }
    let rawAssistantContent = ''
    let streamDone = false
    messages.value.push(assistantMsg)
    saveStreamDraft()
    await scrollToBottom({ force: true })

    try {
      await chatWithAIStream(payload, (event) => {
        if (event.type === 'ping') {
          return
        }
        if (event.type === 'meta') {
          if (event.sessionId) sessionId.value = event.sessionId
          if (event.message?.id) assistantMsg.id = event.message.id
          if (event.message?.status) assistantMsg.status = event.message.status
          if (event.message?.contextRef) assistantMsg.contextRef = event.message.contextRef
          if (event.message?.status === 'generating') assistantMsg.streaming = true
          assistantMsg.thinkingSteps = event.thinkingSteps || []
          if (sessionId.value && assistantMsg.streaming) {
            startGeneratingPoll()
          }
          saveStreamDraft()
          void loadSessions()
          void scrollToBottom()
          return
        }
        if (event.type === 'delta') {
          hasStreamDelta = true
          rawAssistantContent += event.delta || ''
          assistantMsg.content = stripThinkingMarkup(rawAssistantContent)
          assistantMsg.status = 'generating'
          assistantMsg.streaming = true
          saveStreamDraft()
          void scrollToBottom()
          return
        }
        if (event.type === 'done') {
          streamDone = true
          if (event.sessionId) sessionId.value = event.sessionId
          if (event.message?.id) assistantMsg.id = event.message.id
          const doneStatus = event.message?.status
          assistantMsg.status = doneStatus && !isGeneratingStatus(doneStatus) ? doneStatus : 'success'
          if (event.message?.contextRef) assistantMsg.contextRef = event.message.contextRef
          if (event.answer) assistantMsg.content = stripThinkingMarkup(event.answer)
          assistantMsg.finishReason = event.finishReason
          if (isLengthFinish(event.finishReason) || event.message?.status === 'truncated') {
            markTruncated(assistantMsg)
            ElMessage.warning('回答被模型最大输出长度截断，可点击继续生成')
          }
          assistantMsg.streaming = false
          clearStreamDraft()
          stopGeneratingPoll()
          void loadSessions()
          void scrollToBottom()
          return
        }
        if (event.type === 'error') {
          if (event.message?.id) assistantMsg.id = event.message.id
          if (event.message?.status) assistantMsg.status = event.message.status
          if (event.message?.error) assistantMsg.error = event.message.error
          if (event.message?.content) assistantMsg.content = stripThinkingMarkup(event.message.content)
          assistantMsg.streaming = false
          if (!isTerminalMessageStatus(assistantMsg.status)) {
            assistantMsg.status = 'interrupted'
          }
          stopGeneratingPoll()
          const error = new Error(event.error || 'AI 流式问答失败')
          ;(error as any).streamEvent = event
          throw error
        }
      }, { signal: controller.signal })
      if (!streamDone && hasStreamDelta) {
        await refreshActiveSession()
        if (hasGeneratingMessage()) {
          startGeneratingPoll()
        } else {
          assistantMsg.content = stripThinkingMarkup(rawAssistantContent)
          assistantMsg.status = 'interrupted'
          markIncomplete(assistantMsg, '流式连接已结束但服务端没有继续生成，已保留已收到内容，可以继续生成。')
          saveStreamDraft()
          void loadSessions()
        }
      }
      assistantMsg.streaming = false
    } catch (streamError: any) {
      if (streamError?.name === 'AbortError') {
        assistantMsg.streaming = false
        if (completedBySessionRefresh.value || isTerminalMessageStatus(assistantMsg.status)) {
          completedBySessionRefresh.value = false
          return
        }
        if (assistantMsg.content) {
          assistantMsg.status = 'interrupted'
          markIncomplete(assistantMsg, '生成被中断，当前内容已作为本地草稿保留，可以继续生成。')
          saveStreamDraft()
        }
        return
      }
      if (hasStreamDelta) {
        assistantMsg.streaming = false
        assistantMsg.status = 'interrupted'
        const streamReason = streamError?.streamEvent?.error || streamError?.message || 'AI 流式生成中断'
        markIncomplete(assistantMsg, `${streamReason}，已保留已收到内容，可以继续生成。`)
        saveStreamDraft()
        ElMessage.error(streamReason)
        void loadSessions()
        return
      }
      const res = await chatWithAI(payload)
      sessionId.value = res.sessionId
      if (res.message?.id) assistantMsg.id = res.message.id
      if (res.message?.status) assistantMsg.status = res.message.status
      assistantMsg.content = res.answer
      assistantMsg.thinkingSteps = res.thinkingSteps || []
      assistantMsg.finishReason = res.finishReason
      if (isLengthFinish(res.finishReason) || res.message?.status === 'truncated') {
        markTruncated(assistantMsg)
        ElMessage.warning('回答被模型最大输出长度截断，可点击继续生成')
      }
      assistantMsg.streaming = false
      clearStreamDraft()
      await loadSessions()
      await scrollToBottom()
    }
  } finally {
    if (activeController.value === controller) {
      activeController.value = undefined
    }
    sending.value = false
  }
}

onMounted(async () => {
  restoreStreamDraft()
  await loadSessions()
  if (sessionId.value) {
    await refreshActiveSession()
    if (hasGeneratingMessage()) {
      startGeneratingPoll()
    }
  }
  void scrollToBottom({ force: true })
})

onBeforeUnmount(() => {
  stopGeneratingPoll()
})
</script>

<style scoped>
.assistant-page {
  min-height: 100%;
}

.assistant-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 26px;
  border-radius: 20px;
  background: #111827;
  color: #fff;
  box-shadow: 0 22px 45px rgba(17, 24, 39, 0.18);
  margin-bottom: 18px;
}

.hero-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex: 0 0 auto;
  min-width: 420px;
}

.assistant-hero h2 {
  margin: 4px 0 8px;
  font-size: 28px;
}

.assistant-hero p {
  margin: 0;
  color: #cbd5e1;
}

.eyebrow {
  color: #93c5fd;
  font-size: 12px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.assistant-layout {
  display: grid;
  grid-template-columns: 290px 1fr;
  gap: 18px;
  min-height: 0;
}

.side-panel,
.chat-panel {
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid #e5e7eb;
  border-radius: 18px;
  box-shadow: 0 12px 32px rgba(31, 45, 61, 0.08);
}

.side-panel {
  padding: 16px;
  height: calc(100vh - 220px);
  min-height: 560px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-title {
  font-weight: 800;
  color: #111827;
  margin-bottom: 4px;
}

.session-sidebar-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  padding-bottom: 14px;
  margin-bottom: 12px;
  border-bottom: 1px solid #eef2f7;
}

.session-sidebar-head p {
  margin: 0;
  color: #94a3b8;
  font-size: 12px;
}

.side-new-button {
  width: 34px;
  height: 34px;
  padding: 0;
  border-radius: 11px;
  background: #111827;
  border-color: #111827;
  color: #fff;
}

.session-list-wrap {
  flex: 1;
}

.session-empty {
  min-height: 260px;
  border: 1px dashed #cbd5e1;
  border-radius: 16px;
  background: #f8fafc;
  color: #64748b;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 20px;
  text-align: center;
  font-size: 13px;
  line-height: 1.6;
}

.session-empty .el-icon {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  background: #e2e8f0;
  color: #334155;
  font-size: 20px;
}

.session-item {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  text-align: left;
  border: 1px solid #e5e7eb;
  background: #fff;
  border-radius: 14px;
  padding: 11px 8px 11px 12px;
  color: #334155;
  margin-bottom: 10px;
  cursor: pointer;
  transition: all 0.16s ease;
}

.session-item:hover {
  border-color: #cbd5e1;
  background: #f8fafc;
  transform: translateY(-1px);
}

.session-item.active {
  border-color: #111827;
  background: #111827;
  color: #fff;
  box-shadow: 0 12px 24px rgba(17, 24, 39, 0.14);
}

.session-item-main {
  flex: 1;
  min-width: 0;
}

.session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 800;
  font-size: 14px;
  color: inherit;
}

.session-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  color: #94a3b8;
  font-size: 12px;
}

.session-item.active .session-meta {
  color: rgba(255, 255, 255, 0.68);
}

.session-delete {
  width: 30px;
  height: 30px;
  padding: 0;
  flex: 0 0 30px;
  border-radius: 10px;
  color: #94a3b8;
  opacity: 0.35;
}

.session-item:hover .session-delete {
  opacity: 1;
}

.session-delete:hover {
  color: #dc2626;
  background: #fee2e2;
}

.session-item.active .session-delete {
  color: rgba(255, 255, 255, 0.76);
}

.session-item.active .session-delete:hover {
  color: #fff;
  background: rgba(239, 68, 68, 0.38);
}

.chat-panel {
  height: calc(100vh - 220px);
  min-height: 560px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.message-wrap {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px;
}

.empty-state {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #64748b;
  text-align: center;
}

.empty-icon {
  width: 72px;
  height: 72px;
  border-radius: 24px;
  background: #eef2ff;
  color: #3730a3;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
}

.message-row {
  display: flex;
  gap: 12px;
  margin-bottom: 18px;
}

.message-row.user {
  flex-direction: row-reverse;
}

.avatar {
  width: 38px;
  height: 38px;
  border-radius: 13px;
  background: #111827;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.message-row.user .avatar {
  background: #2563eb;
}

.bubble {
  max-width: 78%;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  padding: 12px 14px;
}

.message-row.user .bubble {
  background: #eff6ff;
  border-color: #bfdbfe;
}

.bubble-meta {
  color: #64748b;
  font-size: 12px;
  margin-bottom: 8px;
}

.thinking-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  padding: 10px 12px;
  margin-bottom: 10px;
}

.thinking-title {
  color: #111827;
  font-size: 13px;
  font-weight: 800;
  margin-bottom: 8px;
}

.thinking-step {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  color: #475569;
  font-size: 12px;
  line-height: 1.6;
}

.thinking-step span {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: #111827;
  flex-shrink: 0;
  margin-top: 7px;
}

.streaming-placeholder {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  color: #64748b;
  font-size: 13px;
  line-height: 1.7;
}

.streaming-placeholder span {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #111827;
  animation: pulse-dot 1s ease-in-out infinite;
}

.streaming-inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  padding: 7px 10px;
  border-radius: 999px;
  background: #eef2f7;
  color: #475569;
  font-size: 12px;
  font-weight: 700;
}

.streaming-inline span {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: #111827;
  animation: pulse-dot 1s ease-in-out infinite;
}

.truncate-tip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
  padding: 10px 12px;
  border: 1px solid #fde68a;
  border-radius: 12px;
  background: #fffbeb;
  color: #92400e;
  font-size: 13px;
}

.continue-button {
  flex: 0 0 auto;
  border-color: #111827;
  background: #111827;
  color: #fff;
}

.answer-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}

.answer-action-button {
  border-color: #cbd5e1;
  background: #fff;
  color: #334155;
  border-radius: 10px;
  font-weight: 700;
}

.answer-action-button:hover {
  border-color: #111827;
  color: #111827;
  background: #f8fafc;
}

.answer-action-button.primary {
  border-color: #111827;
  background: #111827;
  color: #fff;
}

.answer-action-button.primary:hover {
  border-color: #111827;
  background: #020617;
  color: #fff;
}

.bubble pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  line-height: 1.7;
  color: #111827;
}

.input-area {
  flex: 0 0 auto;
  position: sticky;
  bottom: 0;
  z-index: 5;
  border-top: 1px solid #e5e7eb;
  padding: 16px;
  background: #fff;
  box-shadow: 0 -12px 28px rgba(15, 23, 42, 0.06);
}

.input-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
  color: #94a3b8;
  font-size: 12px;
}

.black-button {
  background: #111827;
  border-color: #111827;
  color: #fff;
}

.stop-button {
  background: #fff;
  border-color: #cbd5e1;
  color: #111827;
}

.stop-button:hover {
  border-color: #111827;
  color: #111827;
}

@keyframes pulse-dot {
  0%,
  100% {
    opacity: 0.35;
    transform: scale(0.85);
  }

  50% {
    opacity: 1;
    transform: scale(1);
  }
}

@media (max-width: 960px) {
  .assistant-layout {
    grid-template-columns: 1fr;
  }

  .assistant-hero {
    align-items: flex-start;
    gap: 16px;
    flex-direction: column;
  }

  .hero-actions {
    width: 100%;
    min-width: 0;
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .side-panel {
    height: auto;
    min-height: 260px;
    max-height: 360px;
  }

  .chat-panel {
    height: calc(100vh - 260px);
    min-height: 520px;
  }
}
</style>

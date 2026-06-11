<template>
  <section v-if="series.length" class="callback-chart-shell">
    <div class="callback-chart-head">
      <div>
        <span>指标趋势</span>
        <strong>{{ chartTitle }}</strong>
      </div>
      <div class="callback-chart-stats">
        <span>最新值 <b>{{ latestValueText }}</b></span>
        <span>序列 <b>{{ series.length }}</b></span>
        <span v-if="thresholdText">阈值 <b>{{ thresholdText }}</b></span>
      </div>
    </div>
    <div ref="chartRef" class="callback-chart"></div>
    <div class="callback-chart-foot">
      <span>{{ timeRangeText }}</span>
      <span>{{ unitMeta.axisName }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { MonitorAlertCallbackResult, MonitorAlertEvent } from '@/api/monitor-datasource'

interface CallbackPoint {
  time: number
  value: number
}

interface CallbackSeries {
  name: string
  points: CallbackPoint[]
}

interface UnitMeta {
  key: 'percent' | 'bytes' | 'seconds' | 'milliseconds' | 'count' | 'none'
  label: string
  axisName: string
}

const props = defineProps<{
  item: MonitorAlertCallbackResult
  event?: MonitorAlertEvent
}>()

const colors = ['#2563eb', '#16a34a', '#d97706', '#dc2626', '#7c3aed', '#0891b2', '#db2777', '#4b5563']
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null

const renderedQuery = computed(() => props.item.renderedQuery || props.item.query || '')
const chartTitle = computed(() => props.item.name || '回调查询')
const series = computed(() => extractCallbackSeries(props.item))
const unitMeta = computed(() => inferMetricUnit(`${chartTitle.value} ${renderedQuery.value}`))
const hasLineData = computed(() => series.value.some(item => item.points.length > 1))
const latestValue = computed(() => {
  const values = series.value
    .map(item => item.points[item.points.length - 1]?.value)
    .filter((value): value is number => typeof value === 'number' && Number.isFinite(value))
  if (!values.length) return undefined
  return hasLineData.value ? Math.max(...values) : values[0]
})
const latestValueText = computed(() => formatMetricValue(latestValue.value, unitMeta.value))
const thresholdValue = computed(() => {
  const value = Number(props.event?.threshold)
  return Number.isFinite(value) ? value : undefined
})
const thresholdText = computed(() => {
  if (thresholdValue.value === undefined) return ''
  return `${conditionText(props.event?.condition)} ${formatMetricValue(thresholdValue.value, unitMeta.value)}`
})
const timeRangeText = computed(() => {
  const times = series.value.flatMap(item => item.points.map(point => point.time))
  if (!times.length) return '暂无时间范围'
  const min = Math.min(...times)
  const max = Math.max(...times)
  if (min === max) return formatAxisTime(min)
  return `${formatAxisTime(min)} - ${formatAxisTime(max)}`
})

const renderChart = async () => {
  await nextTick()
  if (!chartRef.value || !series.value.length) return
  if (!chart) {
    chart = echarts.init(chartRef.value)
  }
  chart.setOption(hasLineData.value ? buildLineOption() : buildBarOption(), true)
  chart.resize()
}

const buildLineOption = (): echarts.EChartsOption => {
  const lineSeries = series.value.filter(item => item.points.length > 1).slice(0, 8)
  return {
    color: colors,
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#ffffff',
      borderColor: '#d8dee9',
      borderWidth: 1,
      textStyle: { color: '#344054', fontSize: 12 },
      extraCssText: 'box-shadow:0 14px 36px rgba(15,23,42,.12);border-radius:8px;',
      formatter: formatLineTooltip
    },
    legend: {
      type: 'scroll',
      top: 0,
      right: 8,
      itemWidth: 10,
      itemHeight: 8,
      textStyle: { color: '#667085', fontSize: 12 }
    },
    grid: { left: 64, right: 34, top: 52, bottom: 72, containLabel: true },
    xAxis: {
      type: 'time',
      axisLine: { lineStyle: { color: '#d8dee9' } },
      axisTick: { show: false },
      axisLabel: {
        color: '#667085',
        fontSize: 12,
        hideOverlap: true,
        margin: 14,
        formatter: (value: number) => formatAxisTime(value)
      }
    },
    yAxis: {
      type: 'value',
      name: unitMeta.value.label,
      nameTextStyle: { color: '#667085', fontSize: 12, padding: [0, 0, 0, 4] },
      axisLabel: { color: '#667085', fontSize: 12, formatter: (value: number) => formatMetricValue(value, unitMeta.value) },
      splitLine: { lineStyle: { color: '#edf1f7', type: 'dashed' } }
    },
    series: lineSeries.map((item, index) => {
      const option: Record<string, unknown> = {
        name: item.name,
        type: 'line',
        smooth: true,
        showSymbol: false,
        symbolSize: 5,
        connectNulls: true,
        data: item.points.map(point => [normalizeTime(point.time), point.value]),
        lineStyle: { width: 2.2 },
        areaStyle: index === 0 ? { opacity: 0.08 } : undefined
      }
      if (index === 0 && thresholdValue.value !== undefined) {
        option.markLine = {
          symbol: 'none',
          silent: true,
          label: {
            show: false
          },
          lineStyle: { color: '#dc2626', type: 'dashed', width: 1.5 },
          data: [{ yAxis: thresholdValue.value }]
        }
      }
      return option
    })
  }
}

const buildBarOption = (): echarts.EChartsOption => {
  const barItems = series.value.slice(0, 18).map((item, index) => ({
    name: item.name || `Series ${index + 1}`,
    value: item.points[item.points.length - 1]?.value ?? 0
  }))
  return {
    color: ['#2563eb'],
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#ffffff',
      borderColor: '#d8dee9',
      borderWidth: 1,
      textStyle: { color: '#344054', fontSize: 12 },
      extraCssText: 'box-shadow:0 14px 36px rgba(15,23,42,.12);border-radius:8px;',
      valueFormatter: (value: unknown) => formatMetricValue(Number(value), unitMeta.value)
    },
    grid: { left: 64, right: 34, top: 36, bottom: 86, containLabel: true },
    xAxis: {
      type: 'category',
      data: barItems.map(item => item.name),
      axisLine: { lineStyle: { color: '#d8dee9' } },
      axisTick: { show: false },
      axisLabel: {
        color: '#667085',
        fontSize: 12,
        interval: 0,
        rotate: 22,
        hideOverlap: true,
        overflow: 'truncate',
        width: 96
      }
    },
    yAxis: {
      type: 'value',
      name: unitMeta.value.label,
      nameTextStyle: { color: '#667085', fontSize: 12 },
      axisLabel: { color: '#667085', fontSize: 12, formatter: (value: number) => formatMetricValue(value, unitMeta.value) },
      splitLine: { lineStyle: { color: '#edf1f7', type: 'dashed' } }
    },
    series: [{
      name: chartTitle.value,
      type: 'bar',
      barMaxWidth: 34,
      data: barItems.map(item => item.value),
      itemStyle: { borderRadius: [5, 5, 0, 0] },
      markLine: thresholdValue.value === undefined ? undefined : {
        symbol: 'none',
        silent: true,
        label: { show: false },
        lineStyle: { color: '#dc2626', type: 'dashed', width: 1.5 },
        data: [{ yAxis: thresholdValue.value }]
      }
    }]
  }
}

const formatLineTooltip = (params: unknown) => {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as any
  const title = formatAxisTime(Number(first?.value?.[0] ?? first?.axisValue))
  const lines = items.map((item: any) => {
    const color = item?.color || '#2563eb'
    const value = Array.isArray(item?.value) ? item.value[1] : item?.value
    return `<div style="display:flex;align-items:center;gap:8px;margin-top:5px;">
      <span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${color};"></span>
      <span style="min-width:0;max-width:260px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${escapeHtml(item?.seriesName || '-')}</span>
      <b style="margin-left:auto;color:#101828;">${formatMetricValue(Number(value), unitMeta.value)}</b>
    </div>`
  })
  return `<div style="font-weight:600;color:#101828;margin-bottom:4px;">${title}</div>${lines.join('')}`
}

const extractCallbackSeries = (item: MonitorAlertCallbackResult): CallbackSeries[] => {
  const raw: any = item.result
  const result = raw?.data?.result || raw?.result || []
  if (!Array.isArray(result)) return []
  return result
    .map((entry: any, index: number) => {
      const values = Array.isArray(entry?.values)
        ? entry.values
        : Array.isArray(entry?.value)
          ? [entry.value]
          : []
      const points = values
        .map((pair: any[]) => {
          const time = Number(pair?.[0])
          const value = Number(pair?.[1])
          if (!Number.isFinite(time) || !Number.isFinite(value)) return undefined
          return { time, value }
        })
        .filter((point: CallbackPoint | undefined): point is CallbackPoint => Boolean(point))
      if (!points.length) return undefined
      return {
        name: callbackSeriesName(entry?.metric, index),
        points
      }
    })
    .filter((item: CallbackSeries | undefined): item is CallbackSeries => Boolean(item))
}

const callbackSeriesName = (metric: Record<string, unknown> | undefined, index: number) => {
  if (!metric || typeof metric !== 'object') return `Series ${index + 1}`
  const preferred = ['instance', 'pod', 'container', 'job', 'namespace', '__name__']
  for (const key of preferred) {
    const value = metric[key]
    if (value !== undefined && value !== null && String(value)) return String(value)
  }
  return Object.entries(metric)
    .slice(0, 2)
    .map(([key, value]) => `${key}=${value}`)
    .join(', ') || `Series ${index + 1}`
}

const inferMetricUnit = (text: string): UnitMeta => {
  const source = text.toLowerCase()
  if (source.includes('%') || source.includes('使用率') || source.includes('利用率') || source.includes('percent') || source.includes('percentage') || source.includes('utilization') || /\*\s*100/.test(source)) {
    return { key: 'percent', label: '%', axisName: '单位：%' }
  }
  if (source.includes('bytes') || /(^|[_\W])byte($|[_\W])/.test(source)) {
    return { key: 'bytes', label: 'bytes', axisName: '单位：bytes' }
  }
  if (source.includes('milliseconds') || source.includes('_ms') || source.includes('毫秒')) {
    return { key: 'milliseconds', label: 'ms', axisName: '单位：ms' }
  }
  if (source.includes('seconds') || source.includes('_seconds') || source.includes('耗时') || source.includes('延迟')) {
    return { key: 'seconds', label: 's', axisName: '单位：s' }
  }
  if (source.includes('count') || source.includes('total') || source.includes('次数')) {
    return { key: 'count', label: 'count', axisName: '单位：count' }
  }
  return { key: 'none', label: 'value', axisName: '单位：数值' }
}

const formatMetricValue = (value: number | undefined, meta: UnitMeta) => {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-'
  if (meta.key === 'percent') return `${formatCompactNumber(value)}%`
  if (meta.key === 'bytes') return formatBytes(value)
  if (meta.key === 'seconds') return `${formatCompactNumber(value)}s`
  if (meta.key === 'milliseconds') return `${formatCompactNumber(value)}ms`
  return formatCompactNumber(value)
}

const formatCompactNumber = (value: number) => {
  if (!Number.isFinite(value)) return '-'
  if (Math.abs(value) >= 1000) return value.toLocaleString('zh-CN', { maximumFractionDigits: 2 })
  return value.toFixed(2).replace(/\.?0+$/, '')
}

const formatBytes = (value: number) => {
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let normalized = Math.abs(value)
  let index = 0
  while (normalized >= 1024 && index < units.length - 1) {
    normalized /= 1024
    index += 1
  }
  const signed = value < 0 ? -normalized : normalized
  return `${formatCompactNumber(signed)} ${units[index]}`
}

const normalizeTime = (time: number) => time > 100000000000 ? time : time * 1000

const formatAxisTime = (time: number) => {
  if (!Number.isFinite(time)) return '-'
  const date = new Date(normalizeTime(time))
  if (Number.isNaN(date.getTime())) return '-'
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  const hour = `${date.getHours()}`.padStart(2, '0')
  const minute = `${date.getMinutes()}`.padStart(2, '0')
  return `${month}-${day} ${hour}:${minute}`
}

const conditionText = (condition?: string) => {
  const map: Record<string, string> = {
    gt: '>',
    gte: '>=',
    lt: '<',
    lte: '<=',
    eq: '=',
    neq: '!='
  }
  return map[condition || ''] || condition || ''
}

const escapeHtml = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const handleResize = () => chart?.resize()

watch(
  () => [props.item.result, props.item.renderedQuery, props.item.query, props.event?.threshold, props.event?.condition],
  () => renderChart(),
  { deep: true }
)

onMounted(() => {
  window.addEventListener('resize', handleResize)
  renderChart()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chart?.dispose()
  chart = null
})
</script>

<style scoped>
.callback-chart-shell {
  margin-top: 12px;
  padding: 12px;
  border: 1px solid #e8edf5;
  border-radius: 8px;
  background: #ffffff;
}

.callback-chart-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 8px;
}

.callback-chart-head span {
  display: block;
  color: #667085;
  font-size: 12px;
  line-height: 1.4;
}

.callback-chart-head strong {
  display: block;
  margin-top: 2px;
  color: #101828;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.35;
}

.callback-chart-stats {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.callback-chart-stats span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 26px;
  padding: 0 9px;
  border: 1px solid #edf1f7;
  border-radius: 999px;
  background: #fafbfc;
  color: #667085;
  font-size: 12px;
}

.callback-chart-stats b {
  color: #101828;
  font-weight: 700;
}

.callback-chart {
  width: 100%;
  height: 300px;
}

.callback-chart-foot {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding-top: 6px;
  color: #98a2b3;
  font-size: 12px;
  line-height: 1.4;
  min-height: 20px;
}

.callback-chart-foot span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 720px) {
  .callback-chart-head {
    flex-direction: column;
  }

  .callback-chart-stats {
    justify-content: flex-start;
  }

  .callback-chart {
    height: 260px;
  }

  .callback-chart-foot {
    flex-direction: column;
  }
}
</style>

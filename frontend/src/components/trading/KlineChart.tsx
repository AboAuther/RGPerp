import { useEffect, useRef } from 'react'
import {
  CandlestickSeries,
  ColorType,
  createChart,
  HistogramSeries,
  LineSeries,
  type CandlestickData,
  type IChartApi,
  type ISeriesApi,
  type UTCTimestamp,
} from 'lightweight-charts'
import type { KlineItem } from '../../types'

interface KlineChartProps {
  data: KlineItem[]
  height?: number
  dark?: boolean
}

export default function KlineChart({ data, height = 360, dark = true }: KlineChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const volumeSeriesRef = useRef<ISeriesApi<'Histogram'> | null>(null)
  const emaSeriesRef = useRef<ISeriesApi<'Line'> | null>(null)

  useEffect(() => {
    if (!containerRef.current || chartRef.current) {
      return
    }

    const chart = createChart(containerRef.current, {
      height,
      layout: {
        background: { type: ColorType.Solid, color: dark ? '#0b1720' : '#ffffff' },
        textColor: dark ? '#93abc0' : '#475467',
      },
      grid: {
        vertLines: { color: dark ? 'rgba(148, 163, 184, 0.08)' : '#f0f2f5' },
        horzLines: { color: dark ? 'rgba(148, 163, 184, 0.08)' : '#f0f2f5' },
      },
      rightPriceScale: {
        borderColor: dark ? '#17303d' : '#e4e7ec',
      },
      timeScale: {
        borderColor: dark ? '#17303d' : '#e4e7ec',
        timeVisible: true,
        secondsVisible: false,
      },
      crosshair: {
        vertLine: { color: dark ? 'rgba(32, 201, 181, 0.25)' : 'rgba(22, 119, 255, 0.2)' },
        horzLine: { color: dark ? 'rgba(32, 201, 181, 0.25)' : 'rgba(22, 119, 255, 0.2)' },
      },
    })

    const series = chart.addSeries(CandlestickSeries, {
      upColor: '#2ec9b0',
      downColor: '#ff6b6b',
      wickUpColor: '#2ec9b0',
      wickDownColor: '#ff6b6b',
      borderVisible: false,
    })
    const emaSeries = chart.addSeries(LineSeries, {
      color: '#fbbf24',
      lineWidth: 2,
      priceLineVisible: false,
      lastValueVisible: false,
    })
    const volumeSeries = chart.addSeries(HistogramSeries, {
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
      lastValueVisible: false,
      priceLineVisible: false,
    })
    chart.priceScale('volume').applyOptions({
      scaleMargins: {
        top: 0.78,
        bottom: 0,
      },
      borderVisible: false,
    })

    chartRef.current = chart
    seriesRef.current = series
    emaSeriesRef.current = emaSeries
    volumeSeriesRef.current = volumeSeries

    const handleResize = () => {
      if (!containerRef.current || !chartRef.current) {
        return
      }
      chartRef.current.applyOptions({ width: containerRef.current.clientWidth })
    }

    handleResize()
    window.addEventListener('resize', handleResize)
    return () => {
      window.removeEventListener('resize', handleResize)
      chart.remove()
      chartRef.current = null
      seriesRef.current = null
      emaSeriesRef.current = null
      volumeSeriesRef.current = null
    }
  }, [dark, height])

  useEffect(() => {
    if (!seriesRef.current || !volumeSeriesRef.current || !emaSeriesRef.current) {
      return
    }
    const formatted: CandlestickData[] = data.map((item) => ({
      time: item.time as UTCTimestamp,
      open: Number(item.open),
      high: Number(item.high),
      low: Number(item.low),
      close: Number(item.close),
    }))
    const volumes = data.map((item) => ({
      time: item.time as UTCTimestamp,
      value: Number(item.volume || 0),
      color: Number(item.close) >= Number(item.open) ? 'rgba(46, 201, 176, 0.45)' : 'rgba(255, 107, 107, 0.45)',
    }))
    const ema = buildEMA(data, 9).map((item) => ({
      time: item.time as UTCTimestamp,
      value: item.value,
    }))
    seriesRef.current.setData(formatted)
    volumeSeriesRef.current.setData(volumes)
    emaSeriesRef.current.setData(ema)
    chartRef.current?.timeScale().fitContent()
  }, [data])

  return <div ref={containerRef} style={{ width: '100%', height }} />
}

function buildEMA(data: KlineItem[], period: number) {
  const multiplier = 2 / (period + 1)
  let previous = 0
  return data.map((item, idx) => {
    const close = Number(item.close)
    if (idx === 0) {
      previous = close
    } else {
      previous = close * multiplier + previous * (1 - multiplier)
    }
    return { time: item.time, value: previous }
  })
}

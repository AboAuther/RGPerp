import { useEffect, useRef } from 'react'
import {
  CandlestickSeries,
  ColorType,
  createChart,
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

    chartRef.current = chart
    seriesRef.current = series

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
    }
  }, [dark, height])

  useEffect(() => {
    if (!seriesRef.current) {
      return
    }
    const formatted: CandlestickData[] = data.map((item) => ({
      time: item.time as UTCTimestamp,
      open: Number(item.open),
      high: Number(item.high),
      low: Number(item.low),
      close: Number(item.close),
    }))
    seriesRef.current.setData(formatted)
    chartRef.current?.timeScale().fitContent()
  }, [data])

  return <div ref={containerRef} style={{ width: '100%', height }} />
}

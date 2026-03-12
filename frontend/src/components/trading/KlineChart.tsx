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
}

export default function KlineChart({ data, height = 360 }: KlineChartProps) {
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
        background: { type: ColorType.Solid, color: '#ffffff' },
        textColor: '#475467',
      },
      grid: {
        vertLines: { color: '#f0f2f5' },
        horzLines: { color: '#f0f2f5' },
      },
      rightPriceScale: {
        borderColor: '#e4e7ec',
      },
      timeScale: {
        borderColor: '#e4e7ec',
        timeVisible: true,
        secondsVisible: false,
      },
    })

    const series = chart.addSeries(CandlestickSeries, {
      upColor: '#16a34a',
      downColor: '#dc2626',
      wickUpColor: '#16a34a',
      wickDownColor: '#dc2626',
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
  }, [height])

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

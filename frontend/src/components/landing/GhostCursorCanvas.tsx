import { useEffect, useRef } from 'react'

type Pointer = {
  x: number
  y: number
}

type GhostCursorCanvasProps = {
  pointer: Pointer
}

export default function GhostCursorCanvas({ pointer }: GhostCursorCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const pointerRef = useRef(pointer)

  useEffect(() => {
    pointerRef.current = pointer
  }, [pointer])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) {
      return
    }

    const context = canvas.getContext('2d')
    if (!context) {
      return
    }

    const trail = Array.from({ length: 20 }, () => ({
      x: pointer.x,
      y: pointer.y,
      radius: 10,
    }))

    let frameId = 0

    const resize = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2)
      canvas.width = Math.floor(window.innerWidth * dpr)
      canvas.height = Math.floor(window.innerHeight * dpr)
      canvas.style.width = `${window.innerWidth}px`
      canvas.style.height = `${window.innerHeight}px`
      context.setTransform(dpr, 0, 0, dpr, 0, 0)
    }

    const render = () => {
      context.clearRect(0, 0, window.innerWidth, window.innerHeight)

      trail[0].x += (pointerRef.current.x - trail[0].x) * 0.17
      trail[0].y += (pointerRef.current.y - trail[0].y) * 0.17

      for (let index = 1; index < trail.length; index += 1) {
        trail[index].x += (trail[index - 1].x - trail[index].x) * 0.18
        trail[index].y += (trail[index - 1].y - trail[index].y) * 0.18
      }

      for (let index = trail.length - 1; index >= 0; index -= 1) {
        const point = trail[index]
        const progress = 1 - index / trail.length
        const radius = 8 + progress * 26

        const gradient = context.createRadialGradient(point.x, point.y, 0, point.x, point.y, radius)
        gradient.addColorStop(0, `rgba(240, 220, 255, ${0.05 + progress * 0.04})`)
        gradient.addColorStop(0.45, `rgba(190, 126, 255, ${0.04 + progress * 0.05})`)
        gradient.addColorStop(1, 'rgba(190, 126, 255, 0)')

        context.fillStyle = gradient
        context.beginPath()
        context.arc(point.x, point.y, radius, 0, Math.PI * 2)
        context.fill()
      }

      context.beginPath()
      context.fillStyle = 'rgba(246, 235, 255, 0.9)'
      context.shadowBlur = 24
      context.shadowColor = 'rgba(214, 160, 255, 0.8)'
      context.arc(trail[0].x, trail[0].y, 5.2, 0, Math.PI * 2)
      context.fill()
      context.shadowBlur = 0

      frameId = window.requestAnimationFrame(render)
    }

    resize()
    frameId = window.requestAnimationFrame(render)
    window.addEventListener('resize', resize)

    return () => {
      window.cancelAnimationFrame(frameId)
      window.removeEventListener('resize', resize)
    }
  }, [])

  return <canvas ref={canvasRef} className="landing-ghost-canvas" aria-hidden />
}

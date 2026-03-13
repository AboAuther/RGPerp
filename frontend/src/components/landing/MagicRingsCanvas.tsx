import { useEffect, useRef } from 'react'

type Pointer = {
  x: number
  y: number
}

type MagicRingsCanvasProps = {
  pointer: Pointer
}

export default function MagicRingsCanvas({ pointer }: MagicRingsCanvasProps) {
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

    let frameId = 0

    const resize = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2)
      const width = Math.floor(canvas.clientWidth * dpr)
      const height = Math.floor(canvas.clientHeight * dpr)

      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width
        canvas.height = height
      }

      context.setTransform(dpr, 0, 0, dpr, 0, 0)
    }

    const render = (time: number) => {
      resize()

      const width = canvas.clientWidth
      const height = canvas.clientHeight
      const centerX = width / 2
      const centerY = height / 2
      const pointerX = (pointerRef.current.x / Math.max(window.innerWidth, 1) - 0.5) * 18
      const pointerY = (pointerRef.current.y / Math.max(window.innerHeight, 1) - 0.5) * 10
      const wobble = Math.sin(time * 0.0008) * 5

      context.clearRect(0, 0, width, height)
      context.fillStyle = 'rgba(3, 6, 14, 0.86)'
      context.fillRect(0, 0, width, height)

      const leftGlow = context.createRadialGradient(width * 0.14, centerY, 10, width * 0.14, centerY, width * 0.28)
      leftGlow.addColorStop(0, 'rgba(205, 110, 255, 0.36)')
      leftGlow.addColorStop(0.35, 'rgba(166, 102, 255, 0.18)')
      leftGlow.addColorStop(1, 'rgba(0, 0, 0, 0)')
      context.fillStyle = leftGlow
      context.fillRect(0, 0, width, height)

      const rightGlow = context.createRadialGradient(width * 0.86, centerY, 10, width * 0.86, centerY, width * 0.28)
      rightGlow.addColorStop(0, 'rgba(205, 110, 255, 0.34)')
      rightGlow.addColorStop(0.35, 'rgba(166, 102, 255, 0.16)')
      rightGlow.addColorStop(1, 'rgba(0, 0, 0, 0)')
      context.fillStyle = rightGlow
      context.fillRect(0, 0, width, height)

      const rings = [
        { rx: width * 0.29, ry: height * 0.47, color: 'rgba(246, 92, 255, 0.82)', glow: 'rgba(246, 92, 255, 0.52)', line: 2.2 },
        { rx: width * 0.41, ry: height * 0.56, color: 'rgba(213, 120, 255, 0.70)', glow: 'rgba(213, 120, 255, 0.40)', line: 2.1 },
        { rx: width * 0.53, ry: height * 0.66, color: 'rgba(179, 150, 255, 0.58)', glow: 'rgba(179, 150, 255, 0.32)', line: 1.8 },
        { rx: width * 0.65, ry: height * 0.76, color: 'rgba(150, 173, 255, 0.42)', glow: 'rgba(150, 173, 255, 0.24)', line: 1.6 },
      ]

      context.save()
      context.translate(centerX + pointerX * 0.6, centerY + pointerY * 0.3)
      context.rotate((pointerX / 220 + wobble / 400) * Math.PI)

      rings.forEach((ring, index) => {
        const rx = ring.rx + index * 2 + wobble
        const ry = ring.ry + index * 3

        context.beginPath()
        context.strokeStyle = ring.color
        context.lineWidth = ring.line
        context.shadowBlur = 34
        context.shadowColor = ring.glow
        context.ellipse(0, 0, rx, ry, 0, 0, Math.PI * 2)
        context.stroke()
      })

      context.restore()

      const centerMask = context.createRadialGradient(centerX, centerY, width * 0.08, centerX, centerY, width * 0.26)
      centerMask.addColorStop(0, 'rgba(0, 0, 0, 0.22)')
      centerMask.addColorStop(0.65, 'rgba(0, 0, 0, 0.88)')
      centerMask.addColorStop(1, 'rgba(0, 0, 0, 1)')
      context.fillStyle = centerMask
      context.beginPath()
      context.ellipse(centerX, centerY, width * 0.22, height * 0.38, 0, 0, Math.PI * 2)
      context.fill()

      context.fillStyle = 'rgba(255, 255, 255, 0.03)'
      context.fillRect(width * 0.48, 0, width * 0.04, height)

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

  return <canvas ref={canvasRef} className="landing-magic-canvas" aria-hidden />
}

declare type GhostCursorProps = {
  className?: string
  style?: import('react').CSSProperties
  trailLength?: number
  inertia?: number
  grainIntensity?: number
  bloomStrength?: number
  bloomRadius?: number
  bloomThreshold?: number
  brightness?: number
  color?: string
  mixBlendMode?: string
  edgeIntensity?: number
  maxDevicePixelRatio?: number
  targetPixels?: number
  fadeDelayMs?: number
  fadeDurationMs?: number
  zIndex?: number
}

declare function GhostCursor(props: GhostCursorProps): import('react/jsx-runtime').JSX.Element

export default GhostCursor

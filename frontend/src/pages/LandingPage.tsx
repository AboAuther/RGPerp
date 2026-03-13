import { Button, Typography } from 'antd'
import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  siBinance,
  siChainlink,
  siCoinbase,
  siEthereum,
  siGitkraken,
  siOptimism,
  siPolygon,
  siSolana,
} from 'simple-icons'
import BlobCursor from '../components/reactbits/BlobCursor'
import MagicRings from '../components/reactbits/MagicRings'
import './LandingPage.css'

type LogoItem = {
  name: string
  hex: string
  path: string
}

type FeatureItem = {
  title: string
  description: string
}

type StatItem = {
  label: string
  value: string
}

const logos: LogoItem[] = [
  { name: siBinance.title, hex: siBinance.hex, path: siBinance.path },
  { name: siCoinbase.title, hex: siCoinbase.hex, path: siCoinbase.path },
  { name: siEthereum.title, hex: siEthereum.hex, path: siEthereum.path },
  { name: siSolana.title, hex: siSolana.hex, path: siSolana.path },
  { name: siPolygon.title, hex: siPolygon.hex, path: siPolygon.path },
  { name: siOptimism.title, hex: siOptimism.hex, path: siOptimism.path },
  { name: siChainlink.title, hex: siChainlink.hex, path: siChainlink.path },
  { name: siGitkraken.title, hex: siGitkraken.hex, path: siGitkraken.path },
]

const stats: StatItem[] = [
  { label: 'Order Processing', value: '< 50ms' },
  { label: 'Supported Markets', value: 'Multi-Symbol' },
  { label: 'Risk Sync', value: 'Real-Time' },
  { label: 'External Hedge', value: 'Delta Neutral' },
]

const features: FeatureItem[] = [
  {
    title: 'Deterministic Risk Engine',
    description: 'Margin checks, reduce-only states, liquidation thresholds, and stale-price protection are enforced in the order path.',
  },
  {
    title: 'Signature Authorized Vault Flows',
    description: 'Deposits are indexed on-chain while withdrawals are protected by operator signatures and replay-safe nonces.',
  },
  {
    title: 'External Hedge Automation',
    description: 'Internal net exposure is mirrored to external venues through task-based hedge execution with drift monitoring.',
  },
  {
    title: 'Realtime Exchange Surfaces',
    description: 'Market data, account state, positions, and trade history are designed for responsive trading interfaces and dashboards.',
  },
]

export default function LandingPage() {
  const navigate = useNavigate()
  const logoTrack = useMemo(() => [...logos, ...logos], [])

  return (
    <div className="landing-root">
      <div className="landing-rings-layer" aria-hidden>
        <MagicRings
          color="#ff38f5"
          colorTwo="#4ee7ff"
          speed={0.85}
          ringCount={8}
          attenuation={10}
          lineThickness={3}
          baseRadius={0.18}
          radiusStep={0.09}
          scaleRate={0.16}
          opacity={1}
          noiseAmount={0.008}
          rotation={0}
          ringGap={1.22}
          fadeIn={0.66}
          fadeOut={0.78}
          followMouse={false}
          mouseInfluence={0}
          hoverScale={1}
          parallax={0}
          clickBurst={false}
        />
      </div>
      <div className="landing-rings-glow-layer" aria-hidden>
        <MagicRings
          color="#ff54f7"
          colorTwo="#77ffe1"
          speed={0.82}
          ringCount={8}
          attenuation={7.5}
          lineThickness={4.6}
          baseRadius={0.18}
          radiusStep={0.09}
          scaleRate={0.16}
          opacity={0.85}
          blur={18}
          noiseAmount={0}
          rotation={0}
          ringGap={1.22}
          fadeIn={0.66}
          fadeOut={0.78}
          followMouse={false}
          mouseInfluence={0}
          hoverScale={1}
          parallax={0}
          clickBurst={false}
        />
      </div>
      <div className="landing-rings-vignette" aria-hidden />
      <BlobCursor
        fillColor="#8b5cff"
        trailCount={3}
        sizes={[48, 92, 64]}
        innerSizes={[16, 28, 18]}
        innerColor="rgba(255,255,255,0.72)"
        opacities={[0.46, 0.22, 0.16]}
        shadowColor="rgba(120, 78, 255, 0.24)"
        shadowBlur={18}
        shadowOffsetX={0}
        shadowOffsetY={0}
        filterStdDeviation={26}
        zIndex={7}
      />
      <div className="landing-noise" aria-hidden />

      <main className="landing-main">
        <section className="landing-hero">
          <div className="landing-copy">
            <Typography.Text className="landing-tag">RG Perp</Typography.Text>
            <Typography.Title level={1} className="landing-title">
              Build, Trade,
              <br />
              Hedge
              <br />
              Engine
            </Typography.Title>
            <Typography.Paragraph className="landing-subtitle">
              Clean perpetual infrastructure with risk control, liquidation, and external hedging.
            </Typography.Paragraph>

            <div className="landing-actions">
              <Button className="landing-launch-btn" size="large" onClick={() => navigate('/app')}>
                Launch App
              </Button>
              <Button className="landing-docs-btn" size="large" type="text" onClick={() => navigate('/docs')}>
                Read Docs
              </Button>
            </div>
          </div>
        </section>

        <section className="landing-logos-wrap" aria-label="partners">
          <Typography.Text className="landing-logos-caption">未被以下品牌赞助：</Typography.Text>
          <div className="landing-logos-track">
            {logoTrack.map((item, idx) => (
              <div key={`${item.name}-${idx}`} className="landing-logo-item">
                <span className="landing-logo-mark" aria-hidden>
                  <svg viewBox="0 0 24 24" role="img">
                    <path d={item.path} fill={`#${item.hex}`} />
                  </svg>
                </span>
                <span>{item.name}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="landing-stats-section">
          {stats.map((item) => (
            <article key={item.label} className="landing-stat-panel">
              <span className="landing-stat-label">{item.label}</span>
              <strong className="landing-stat-value">{item.value}</strong>
            </article>
          ))}
        </section>

        <section className="landing-feature-section">
          <div className="landing-section-copy">
            <Typography.Text className="landing-section-tag">Platform Features</Typography.Text>
            <Typography.Title level={2} className="landing-section-title">
              A landing page with product-grade structure
            </Typography.Title>
            <Typography.Paragraph className="landing-section-subtitle">
              The visual language is now closer to React Bits style: clearer hierarchy, stronger CTA focus, and more
              deliberate animated surfaces around the hero.
            </Typography.Paragraph>
          </div>

          <div className="landing-feature-grid">
            {features.map((item) => (
              <article key={item.title} className="landing-feature-card">
                <span className="landing-feature-kicker">Core Module</span>
                <Typography.Title level={4} className="landing-feature-title">
                  {item.title}
                </Typography.Title>
                <Typography.Paragraph className="landing-feature-description">
                  {item.description}
                </Typography.Paragraph>
              </article>
            ))}
          </div>
        </section>
      </main>
    </div>
  )
}

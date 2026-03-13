import { Button, Typography } from 'antd'
import { useNavigate } from 'react-router-dom'

const sections = [
  {
    title: 'Architecture',
    description: 'Frontend, backend API, trade engine, vault contract, indexer, and hedge worker are separated into clear operational boundaries.',
  },
  {
    title: 'Risk Control',
    description: 'Order acceptance, account state transitions, liquidation paths, and stale price checks are validated inside transactional service flows.',
  },
  {
    title: 'Vault Flow',
    description: 'On-chain deposits are indexed into internal balances while withdrawals require operator signatures and confirmed event reconciliation.',
  },
  {
    title: 'Hedging',
    description: 'Internal net exposure is mapped to external hedge tasks, with drift snapshots and retry-safe execution across adapters.',
  },
]

export default function DocsPage() {
  const navigate = useNavigate()

  return (
    <div
      style={{
        minHeight: '100vh',
        padding: '32px 20px 64px',
        background:
          'radial-gradient(circle at top, rgba(102, 169, 255, 0.08), transparent 28%), linear-gradient(180deg, #040812 0%, #060b13 100%)',
      }}
    >
      <div
        style={{
          width: 'min(1080px, 100%)',
          margin: '0 auto',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 16,
            padding: 16,
            borderRadius: 24,
            border: '1px solid rgba(255,255,255,0.08)',
            background: 'rgba(8, 14, 25, 0.72)',
            backdropFilter: 'blur(18px)',
          }}
        >
          <Typography.Title level={4} style={{ margin: 0, color: '#f5fbff' }}>
            RGPerp Docs
          </Typography.Title>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            <Button onClick={() => navigate('/')}>Home</Button>
            <Button type="primary" onClick={() => navigate('/app')}>
              Launch App
            </Button>
          </div>
        </div>

        <div style={{ marginTop: 48 }}>
          <Typography.Text
            style={{
              color: '#8cb8ff',
              fontFamily: '"IBM Plex Mono", monospace',
              letterSpacing: '0.08em',
            }}
          >
            PRODUCT OVERVIEW
          </Typography.Text>
          <Typography.Title
            level={1}
            style={{
              color: '#f7fbff',
              marginTop: 12,
              marginBottom: 12,
              lineHeight: 1.05,
            }}
          >
            Technical documentation entry
          </Typography.Title>
          <Typography.Paragraph
            style={{
              color: '#90a1b5',
              maxWidth: 720,
              fontSize: 16,
              lineHeight: 1.8,
            }}
          >
            This page acts as a public-facing documentation landing surface for the exchange. It links users into the
            architecture, risk, vault, and hedge domains without exposing internal implementation clutter.
          </Typography.Paragraph>
        </div>

        <div
          style={{
            marginTop: 28,
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
            gap: 16,
          }}
        >
          {sections.map((section) => (
            <article
              key={section.title}
              style={{
                padding: 24,
                borderRadius: 24,
                border: '1px solid rgba(255,255,255,0.08)',
                background: 'linear-gradient(180deg, rgba(255,255,255,0.03), rgba(255,255,255,0.015)), rgba(8, 15, 27, 0.8)',
                boxShadow: '0 18px 50px rgba(0,0,0,0.24)',
              }}
            >
              <Typography.Title level={4} style={{ color: '#f5fbff', marginTop: 0 }}>
                {section.title}
              </Typography.Title>
              <Typography.Paragraph style={{ color: '#8fa0b4', marginBottom: 0, lineHeight: 1.75 }}>
                {section.description}
              </Typography.Paragraph>
            </article>
          ))}
        </div>
      </div>
    </div>
  )
}

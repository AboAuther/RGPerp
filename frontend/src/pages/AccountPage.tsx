import { useEffect, useMemo, useRef, useState } from 'react'
import { SwapOutlined } from '@ant-design/icons'
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Form,
  Input,
  Row,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { encodeFunctionData, parseUnits, toHex } from 'viem'
import { post, get } from '../services/api'
import { useAuthStore } from '../stores/authStore'
import { useThemeStore } from '../stores/themeStore'
import type {
  Account,
  AuthChallenge,
  AuthLogin,
  DepositRecord,
  DepositInfo,
  WithdrawalRequest,
} from '../types'

declare global {
  interface Window {
    ethereum?: {
      request: (args: { method: string; params?: unknown[] }) => Promise<unknown>
    }
  }
}

type DeadlineMode = 'local' | 'unix'

function deadlineUnix(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '-'
  }
  return String(Math.floor(date.getTime() / 1000))
}

function formatDeadline(value: string, mode: DeadlineMode): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  if (mode === 'unix') {
    return deadlineUnix(value)
  }
  return date.toLocaleString()
}

function parseUsdcAmountToUnits(value: string): bigint {
  return parseUnits(value, 6)
}

function formatAmount(value?: string | number | null, precision = 6): string {
  if (value === null || value === undefined || value === '') {
    return '--'
  }
  const num = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(num)) {
    return String(value)
  }
  return num.toLocaleString(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: precision,
  })
}

function shortenMiddle(value: string, left = 10, right = 8): string {
  if (value.length <= left + right + 3) {
    return value
  }
  return `${value.slice(0, left)}...${value.slice(-right)}`
}

async function waitForTxReceipt(hash: string): Promise<void> {
  if (!window.ethereum) {
    throw new Error('wallet not found')
  }
  const timeoutAt = Date.now() + 120_000

  while (Date.now() < timeoutAt) {
    const receipt = await window.ethereum.request({
      method: 'eth_getTransactionReceipt',
      params: [hash],
    }) as { status?: string } | null
    if (receipt) {
      if (receipt.status === '0x0') {
        throw new Error('transaction reverted on chain')
      }
      return
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }
  throw new Error('transaction confirmation timeout')
}

export default function AccountPage() {
  const [withdrawForm] = Form.useForm()
  const [depositForm] = Form.useForm()
  const queryClient = useQueryClient()
  const { token, walletAddress, setAuth, logout, isAuthenticated } = useAuthStore()
  const [messageApi, contextHolder] = message.useMessage()
  const [withdrawSignature, setWithdrawSignature] = useState<WithdrawalRequest | null>(null)
  const [deadlineMode, setDeadlineMode] = useState<DeadlineMode>('local')
  const [depositHint, setDepositHint] = useState<string | null>(null)
  const { mode } = useThemeStore()
  const isDark = mode === 'dark'
  const depositHintTimerRef = useRef<number | null>(null)

  const authenticated = isAuthenticated()

  useEffect(() => {
    return () => {
      if (depositHintTimerRef.current) {
        window.clearTimeout(depositHintTimerRef.current)
      }
    }
  }, [])

  const accountQuery = useQuery({
    queryKey: ['account', token],
    queryFn: async () => (await get<Account>('/account')).data!,
    enabled: authenticated,
  })

  const depositInfoQuery = useQuery({
    queryKey: ['deposit-info', token],
    queryFn: async () => (await get<DepositInfo>('/wallet/deposit-info')).data!,
    enabled: authenticated,
  })

  const withdrawalsQuery = useQuery({
    queryKey: ['withdrawals', token],
    queryFn: async () => {
      const res = await get<{ items: WithdrawalRequest[] }>('/withdrawals')
      return res.data?.items ?? []
    },
    enabled: authenticated,
  })

  const depositsQuery = useQuery({
    queryKey: ['deposits', token],
    queryFn: async () => {
      const res = await get<{ items: DepositRecord[] }>('/deposits')
      return res.data?.items ?? []
    },
    enabled: authenticated,
  })

  const loginMutation = useMutation({
    mutationFn: async () => {
      if (!window.ethereum) {
        throw new Error('wallet not found')
      }

      const accounts = (await window.ethereum.request({
        method: 'eth_requestAccounts',
      })) as string[]
      const selectedAddress = accounts?.[0]
      if (!selectedAddress) {
        throw new Error('wallet account not found')
      }

      const challenge = (
        await post<AuthChallenge>('/auth/challenge', {
          wallet_address: selectedAddress,
          domain: window.location.host,
          chain_id: Number(import.meta.env.VITE_CHAIN_ID || 31337),
        })
      ).data!

      const signature = (await window.ethereum.request({
        method: 'personal_sign',
        params: [challenge.message, selectedAddress],
      })) as string

      const login = (
        await post<AuthLogin>('/auth/login', {
          wallet_address: selectedAddress,
          nonce: challenge.nonce,
          signature,
        })
      ).data!

      return login
    },
    onSuccess: (result) => {
      setAuth(result.token, result.user.wallet_address)
      queryClient.invalidateQueries({ queryKey: ['account'] })
      queryClient.invalidateQueries({ queryKey: ['deposit-info'] })
      queryClient.invalidateQueries({ queryKey: ['deposits'] })
      queryClient.invalidateQueries({ queryKey: ['withdrawals'] })
      void messageApi.success('登录成功')
    },
    onError: (error) => {
      void messageApi.error(error instanceof Error ? error.message : '登录失败')
    },
  })

  const withdrawalMutation = useMutation({
    mutationFn: async (amount: string) => {
      const res = await post<WithdrawalRequest>(
        '/withdrawals',
        { amount },
        { 'X-Idempotency-Key': `wd-${Date.now()}` },
      )
      return res.data!
    },
    onSuccess: (result) => {
      setWithdrawSignature(result)
      withdrawForm.resetFields()
      queryClient.invalidateQueries({ queryKey: ['account'] })
      queryClient.invalidateQueries({ queryKey: ['withdrawals'] })
      void messageApi.success('提现授权已生成')
    },
    onError: (error) => {
      void messageApi.error(error instanceof Error ? error.message : '提现申请失败')
    },
  })

  const depositMutation = useMutation({
    mutationFn: async (amount: string) => {
      if (!window.ethereum) {
        throw new Error('wallet not found')
      }
      const depositInfo = depositInfoQuery.data
      if (!depositInfo?.usdc_address || !depositInfo?.vault_address) {
        throw new Error('deposit config not ready')
      }

      const amountUnits = parseUsdcAmountToUnits(amount)
      if (amountUnits <= 0n) {
        throw new Error('invalid deposit amount')
      }

      const walletAccounts = (await window.ethereum.request({
        method: 'eth_requestAccounts',
      })) as string[]
      const from = walletAccounts[0]
      if (!from) {
        throw new Error('wallet account not found')
      }

      try {
        await window.ethereum.request({
          method: 'wallet_switchEthereumChain',
          params: [{ chainId: toHex(Number(depositInfo.chain_id)) }],
        })
      } catch (_err) {
        // Keep going. Some wallets do not support switch on local chains.
      }

      const approveData = encodeFunctionData({
        abi: [
          {
            type: 'function',
            name: 'approve',
            stateMutability: 'nonpayable',
            inputs: [
              { name: 'spender', type: 'address' },
              { name: 'amount', type: 'uint256' },
            ],
            outputs: [{ name: '', type: 'bool' }],
          },
        ],
        functionName: 'approve',
        args: [depositInfo.vault_address as `0x${string}`, amountUnits],
      })

      const approveTxHash = (await window.ethereum.request({
        method: 'eth_sendTransaction',
        params: [
          {
            from,
            to: depositInfo.usdc_address,
            data: approveData,
          },
        ],
      })) as string
      await waitForTxReceipt(approveTxHash)

      const depositData = encodeFunctionData({
        abi: [
          {
            type: 'function',
            name: 'deposit',
            stateMutability: 'nonpayable',
            inputs: [{ name: 'amount', type: 'uint256' }],
            outputs: [],
          },
        ],
        functionName: 'deposit',
        args: [amountUnits],
      })
      const depositTxHash = (await window.ethereum.request({
        method: 'eth_sendTransaction',
        params: [
          {
            from,
            to: depositInfo.vault_address,
            data: depositData,
          },
        ],
      })) as string
      await waitForTxReceipt(depositTxHash)
      return { approveTxHash, depositTxHash }
    },
    onSuccess: (result) => {
      depositForm.resetFields()
      void messageApi.success(`充值交易已上链: ${result.depositTxHash}`)
      setDepositHint('充值已上链，indexer 正在同步，余额通常会在 3 秒内刷新。')
      if (depositHintTimerRef.current) {
        window.clearTimeout(depositHintTimerRef.current)
      }
      depositHintTimerRef.current = window.setTimeout(() => {
        setDepositHint(null)
      }, 3000)
      queryClient.invalidateQueries({ queryKey: ['account'] })
      queryClient.invalidateQueries({ queryKey: ['deposits'] })
      window.setTimeout(() => {
        void queryClient.invalidateQueries({ queryKey: ['account'] })
        void queryClient.invalidateQueries({ queryKey: ['deposits'] })
      }, 3000)
    },
    onError: (error) => {
      void messageApi.error(error instanceof Error ? error.message : '充值失败')
    },
  })

  const executeWithdrawalMutation = useMutation({
    mutationFn: async (request: WithdrawalRequest) => {
      if (!window.ethereum) {
        throw new Error('wallet not found')
      }
      const depositInfo = depositInfoQuery.data
      if (!depositInfo?.vault_address) {
        throw new Error('vault config not ready')
      }

      const walletAccounts = (await window.ethereum.request({
        method: 'eth_requestAccounts',
      })) as string[]
      const from = walletAccounts[0]
      if (!from) {
        throw new Error('wallet account not found')
      }

      try {
        await window.ethereum.request({
          method: 'wallet_switchEthereumChain',
          params: [{ chainId: toHex(Number(depositInfo.chain_id)) }],
        })
      } catch (_err) {
        // Ignore switch failures for local chains.
      }

      const withdrawData = encodeFunctionData({
        abi: [
          {
            type: 'function',
            name: 'withdraw',
            stateMutability: 'nonpayable',
            inputs: [
              { name: 'amount', type: 'uint256' },
              { name: 'nonce', type: 'uint256' },
              { name: 'deadline', type: 'uint256' },
              { name: 'signature', type: 'bytes' },
            ],
            outputs: [],
          },
        ],
        functionName: 'withdraw',
        args: [
          parseUsdcAmountToUnits(request.amount),
          BigInt(request.nonce),
          BigInt(deadlineUnix(request.deadline)),
          request.signature as `0x${string}`,
        ],
      })

      const txHash = (await window.ethereum.request({
        method: 'eth_sendTransaction',
        params: [
          {
            from,
            to: depositInfo.vault_address,
            data: withdrawData,
          },
        ],
      })) as string
      await waitForTxReceipt(txHash)
      return txHash
    },
    onSuccess: (txHash) => {
      void messageApi.success(`提现交易已上链: ${txHash}`)
      setWithdrawSignature(null)
      queryClient.invalidateQueries({ queryKey: ['account'] })
      queryClient.invalidateQueries({ queryKey: ['withdrawals'] })
      window.setTimeout(() => {
        void queryClient.invalidateQueries({ queryKey: ['account'] })
        void queryClient.invalidateQueries({ queryKey: ['withdrawals'] })
      }, 3000)
    },
    onError: (error) => {
      void messageApi.error(error instanceof Error ? error.message : '提现执行失败')
    },
  })

  const withdrawalColumns = useMemo(
    () => [
      { title: '请求 ID', dataIndex: 'request_id', key: 'request_id' },
      { title: '金额', dataIndex: 'amount', key: 'amount', render: (value: string) => `${formatAmount(value)} USDC` },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => <Tag color={status === 'confirmed' ? 'green' : 'blue'}>{status}</Tag>,
      },
      { title: 'Nonce', dataIndex: 'nonce', key: 'nonce' },
      {
        title: (
          <Space size={6}>
            <Typography.Text>截止时间</Typography.Text>
            <Button
              size="small"
              type="text"
              icon={<SwapOutlined />}
              onClick={() => setDeadlineMode((mode) => (mode === 'local' ? 'unix' : 'local'))}
            />
          </Space>
        ),
        dataIndex: 'deadline',
        key: 'deadline',
        render: (value: string) => <Typography.Text>{formatDeadline(value, deadlineMode)}</Typography.Text>,
      },
      { title: '链上 Tx', dataIndex: 'tx_hash', key: 'tx_hash', render: (value?: string) => value || '-' },
      {
        title: '操作',
        key: 'action',
        render: (_: unknown, record: WithdrawalRequest) =>
          record.status === 'signed' ? (
            <Button
              size="small"
              type="primary"
              ghost={isDark}
              loading={executeWithdrawalMutation.isPending}
              onClick={() => executeWithdrawalMutation.mutate(record)}
            >
              钱包提现
            </Button>
          ) : (
            '-'
          ),
      },
    ],
    [deadlineMode, executeWithdrawalMutation, isDark],
  )

  const depositColumns = useMemo(
    () => [
      {
        title: '充值时间',
        dataIndex: 'created_at',
        key: 'created_at',
        render: (value: string) => new Date(value).toLocaleString(),
      },
      { title: '金额', dataIndex: 'amount', key: 'amount', render: (value: string) => `${formatAmount(value)} USDC` },
      { title: '区块', dataIndex: 'block_number', key: 'block_number' },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => <Tag color={status === 'confirmed' ? 'green' : 'blue'}>{status}</Tag>,
      },
      {
        title: '链上 Tx',
        dataIndex: 'tx_hash',
        key: 'tx_hash',
        render: (value: string) => value || '-',
      },
    ],
    [],
  )

  return (
    <div>
      {contextHolder}
      <Typography.Title level={3}>资产总览</Typography.Title>
      {!authenticated ? (
        <Card>
          <Space direction="vertical" size="middle">
            <Typography.Text type="secondary">
              连接钱包并签名登录后，才能查看账户、充值地址和提现授权。
            </Typography.Text>
            <Button type="primary" loading={loginMutation.isPending} onClick={() => loginMutation.mutate()}>
              连接钱包并登录
            </Button>
          </Space>
        </Card>
      ) : null}
      <Row gutter={[16, 16]} align="stretch">
        <Col xs={24} md={8} style={{ display: 'flex' }}>
          <Card
            title="账户余额"
            className="rg-glass-card"
            style={{ width: '100%', height: '100%' }}
            styles={{ body: { display: 'flex', flexDirection: 'column', gap: 18 } }}
          >
            <div
              style={{
                borderRadius: 16,
                padding: 18,
                background: isDark
                  ? 'linear-gradient(135deg, rgba(46,201,176,0.16) 0%, rgba(21,37,48,0.9) 100%)'
                  : 'linear-gradient(135deg, rgba(22,119,255,0.10) 0%, rgba(255,255,255,0.92) 100%)',
                border: `1px solid ${isDark ? '#1d3d4c' : '#dbeafe'}`,
              }}
            >
              <Typography.Text type="secondary">钱包地址</Typography.Text>
              <Typography.Paragraph className="rg-mono" style={{ fontSize: 18, marginBottom: 0, marginTop: 6 }}>
                {walletAddress}
              </Typography.Paragraph>
            </div>

            <Row gutter={[12, 12]}>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="可用余额" value={formatAmount(accountQuery.data?.available_balance)} suffix="USDC" />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="账户权益" value={formatAmount(accountQuery.data?.equity)} suffix="USDC" />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="锁定余额" value={formatAmount(accountQuery.data?.locked_balance)} suffix="USDC" />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="可提现" value={formatAmount(accountQuery.data?.withdrawable_balance)} suffix="USDC" />
                </Card>
              </Col>
            </Row>

            <Descriptions column={1} size="small">
              <Descriptions.Item label="资产">{accountQuery.data?.asset ?? 'USDC'}</Descriptions.Item>
              <Descriptions.Item label="待提现占用">{formatAmount(accountQuery.data?.pending_withdrawal)} USDC</Descriptions.Item>
            </Descriptions>

            <div style={{ marginTop: 'auto' }}>
              <Button onClick={() => logout()}>退出登录</Button>
            </div>
          </Card>
        </Col>
        <Col xs={24} md={8} style={{ display: 'flex' }}>
          <Card
            title="充值"
            className="rg-glass-card"
            style={{ width: '100%', height: '100%' }}
            styles={{ body: { display: 'flex', flexDirection: 'column', height: '100%' } }}
          >
            <Descriptions column={1} size="small">
              <Descriptions.Item label="链 ID">{depositInfoQuery.data?.chain_id ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="Vault">{depositInfoQuery.data?.vault_address ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="USDC">{depositInfoQuery.data?.usdc_address ?? '-'}</Descriptions.Item>
            </Descriptions>
            <Form
              form={depositForm}
              layout="vertical"
              style={{ marginTop: 16 }}
              onFinish={(values: { amount: string }) => depositMutation.mutate(values.amount)}
            >
              <Form.Item
                label="充值金额 (USDC)"
                name="amount"
                rules={[{ required: true, message: '请输入充值金额' }]}
              >
                <Input placeholder="例如 10" />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={depositMutation.isPending}>
                MetaMask 充值 (approve + deposit)
              </Button>
            </Form>
            {depositHint ? (
              <Alert style={{ marginTop: 'auto' }} type="info" showIcon message={depositHint} />
            ) : (
              <div style={{ marginTop: 'auto', minHeight: 52 }} />
            )}
          </Card>
        </Col>
        <Col xs={24} md={8} style={{ display: 'flex' }}>
          <Card
            title="提现"
            className="rg-glass-card"
            style={{ width: '100%', height: '100%' }}
            styles={{ body: { display: 'flex', flexDirection: 'column', gap: 16, height: '100%' } }}
          >
            <Form
              form={withdrawForm}
              layout="vertical"
              onFinish={(values: { amount: string }) => withdrawalMutation.mutate(values.amount)}
            >
              <Form.Item
                label="提现金额 (USDC)"
                name="amount"
                rules={[{ required: true, message: '请输入提现金额' }]}
              >
                <Input placeholder="例如 10" />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={withdrawalMutation.isPending}>
                生成提现授权
              </Button>
            </Form>

            {withdrawSignature ? (
              <div
                style={{
                  borderRadius: 14,
                  padding: 16,
                  background: isDark ? 'rgba(15, 31, 40, 0.9)' : '#f8fbff',
                  border: `1px solid ${isDark ? '#1b3b49' : '#dbeafe'}`,
                }}
              >
                <Space direction="vertical" size={10} style={{ width: '100%' }}>
                  <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
                    <Typography.Text strong>提现授权已生成</Typography.Text>
                    <Button size="small" icon={<SwapOutlined />} onClick={() => setDeadlineMode((mode) => (mode === 'local' ? 'unix' : 'local'))} />
                  </Space>
                  <Descriptions size="small" column={1}>
                    <Descriptions.Item label="Request ID">{withdrawSignature.request_id}</Descriptions.Item>
                    <Descriptions.Item label="Amount">{formatAmount(withdrawSignature.amount)} USDC</Descriptions.Item>
                    <Descriptions.Item label="Nonce">{withdrawSignature.nonce}</Descriptions.Item>
                    <Descriptions.Item label="Deadline">
                      {formatDeadline(withdrawSignature.deadline, deadlineMode)}
                    </Descriptions.Item>
                    <Descriptions.Item label="Signature">
                      <Typography.Text
                        className="rg-mono"
                        copyable={{ text: withdrawSignature.signature }}
                      >
                        {shortenMiddle(withdrawSignature.signature, 14, 10)}
                      </Typography.Text>
                    </Descriptions.Item>
                  </Descriptions>
                  <Button
                    type="primary"
                    ghost={isDark}
                    loading={executeWithdrawalMutation.isPending}
                    onClick={() => executeWithdrawalMutation.mutate(withdrawSignature)}
                  >
                    钱包确认提现
                  </Button>
                </Space>
              </div>
            ) : null}
            <Divider style={{ margin: 0 }} />
            <Typography.Text type="secondary" style={{ marginTop: 'auto' }}>
              先生成授权，再通过钱包确认发起链上提现。
            </Typography.Text>
          </Card>
        </Col>
      </Row>

      {authenticated ? (
        depositsQuery.isError ? (
          <Alert
            style={{ marginTop: 24 }}
            type="error"
            showIcon
            message="充值记录加载失败"
            description="请检查后端服务是否已重启到最新版本，且 `/api/v1/deposits` 路由可用。"
          />
        ) : (
        <Card style={{ marginTop: 24 }} title="充值记录" className="rg-glass-card">
          <Table
            rowKey={(record) => `${record.tx_hash}-${record.log_index}`}
            loading={depositsQuery.isLoading}
            dataSource={depositsQuery.data ?? []}
            columns={depositColumns}
            pagination={false}
            scroll={{ x: 960 }}
          />
        </Card>
        )
      ) : null}

      {authenticated ? (
        withdrawalsQuery.isError ? (
          <Alert
            style={{ marginTop: 24 }}
            type="error"
            showIcon
            message="提现记录加载失败"
            description="请检查后端服务状态和当前登录 token 是否有效。"
          />
        ) : (
        <Card style={{ marginTop: 24 }} title="提现记录" className="rg-glass-card">
          <Table
            rowKey="request_id"
            loading={withdrawalsQuery.isLoading}
            dataSource={withdrawalsQuery.data ?? []}
            columns={withdrawalColumns}
            pagination={false}
            scroll={{ x: 960 }}
          />
        </Card>
        )
      ) : null}
    </div>
  )
}

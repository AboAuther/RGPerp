import { useMemo, useState } from 'react'
import { SwapOutlined } from '@ant-design/icons'
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  Row,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { encodeFunctionData, parseUnits, toHex } from 'viem'
import { post, get } from '../services/api'
import { useAuthStore } from '../stores/authStore'
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

async function waitForTxReceipt(hash: string): Promise<void> {
  if (!window.ethereum) {
    throw new Error('wallet not found')
  }
  const timeoutAt = Date.now() + 120_000

  while (Date.now() < timeoutAt) {
    const receipt = await window.ethereum.request({
      method: 'eth_getTransactionReceipt',
      params: [hash],
    })
    if (receipt) {
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

  const authenticated = isAuthenticated()

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
      queryClient.invalidateQueries({ queryKey: ['account'] })
      queryClient.invalidateQueries({ queryKey: ['deposits'] })
      setTimeout(() => {
        void queryClient.invalidateQueries({ queryKey: ['account'] })
        void queryClient.invalidateQueries({ queryKey: ['deposits'] })
      }, 6000)
    },
    onError: (error) => {
      void messageApi.error(error instanceof Error ? error.message : '充值失败')
    },
  })

  const withdrawalColumns = useMemo(
    () => [
      { title: '请求 ID', dataIndex: 'request_id', key: 'request_id' },
      { title: '金额', dataIndex: 'amount', key: 'amount' },
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
            >
              {deadlineMode === 'local' ? '看时间戳' : '看本地时间'}
            </Button>
          </Space>
        ),
        dataIndex: 'deadline',
        key: 'deadline',
        render: (value: string) => <Typography.Text>{formatDeadline(value, deadlineMode)}</Typography.Text>,
      },
      { title: '链上 Tx', dataIndex: 'tx_hash', key: 'tx_hash', render: (value?: string) => value || '-' },
    ],
    [deadlineMode],
  )

  const depositColumns = useMemo(
    () => [
      {
        title: '充值时间',
        dataIndex: 'created_at',
        key: 'created_at',
        render: (value: string) => new Date(value).toLocaleString(),
      },
      { title: '金额', dataIndex: 'amount', key: 'amount' },
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
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <Card title="账户余额">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="钱包地址">{walletAddress}</Descriptions.Item>
              <Descriptions.Item label="资产">{accountQuery.data?.asset ?? 'USDC'}</Descriptions.Item>
              <Descriptions.Item label="可用余额">{accountQuery.data?.available_balance ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="锁定余额">{accountQuery.data?.locked_balance ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="待提现占用">{accountQuery.data?.pending_withdrawal ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="可提现余额">{accountQuery.data?.withdrawable_balance ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="账户权益">{accountQuery.data?.equity ?? '-'}</Descriptions.Item>
            </Descriptions>
            <Space style={{ marginTop: 16 }}>
              <Button onClick={() => logout()}>退出登录</Button>
            </Space>
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card title="充值">
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
            <Alert
              style={{ marginTop: 16 }}
              type="info"
              showIcon
              message="充值上链后，indexer 入账通常需要 3-6 秒，请稍后刷新余额。"
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card title="提现">
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
              <Alert
                style={{ marginTop: 16 }}
                type="success"
                showIcon
                message="提现授权已生成"
                description={
                  <Space direction="vertical" size={4}>
                    <Typography.Text>Request ID: {withdrawSignature.request_id}</Typography.Text>
                    <Typography.Text>Nonce: {withdrawSignature.nonce}</Typography.Text>
                    <Space>
                      <Typography.Text>
                        Deadline: {formatDeadline(withdrawSignature.deadline, deadlineMode)}
                      </Typography.Text>
                      <Button
                        size="small"
                        onClick={() =>
                          setDeadlineMode((mode) => (mode === 'local' ? 'unix' : 'local'))
                        }
                      >
                        {deadlineMode === 'local' ? '看时间戳' : '看本地时间'}
                      </Button>
                    </Space>
                    <Typography.Text type="secondary">
                      UTC: {new Date(withdrawSignature.deadline).toISOString()}
                    </Typography.Text>
                    <Typography.Text type="secondary">
                      Unix: {deadlineUnix(withdrawSignature.deadline)}
                    </Typography.Text>
                    <Typography.Text copyable={{ text: withdrawSignature.signature }}>
                      Signature: {withdrawSignature.signature}
                    </Typography.Text>
                  </Space>
                }
              />
            ) : null}
          </Card>
        </Col>
      </Row>

      {authenticated ? (
        <Card style={{ marginTop: 24 }} title="充值记录">
          <Table
            rowKey={(record) => `${record.tx_hash}-${record.log_index}`}
            loading={depositsQuery.isLoading}
            dataSource={depositsQuery.data ?? []}
            columns={depositColumns}
            pagination={false}
            scroll={{ x: 960 }}
          />
        </Card>
      ) : null}

      {authenticated ? (
        <Card style={{ marginTop: 24 }} title="提现记录">
          <Table
            rowKey="request_id"
            loading={withdrawalsQuery.isLoading}
            dataSource={withdrawalsQuery.data ?? []}
            columns={withdrawalColumns}
            pagination={false}
            scroll={{ x: 960 }}
          />
        </Card>
      ) : null}
    </div>
  )
}

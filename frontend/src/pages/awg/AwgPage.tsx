import { useEffect, useMemo, useState } from 'react';
import { Alert, Badge, Button, Card, Col, ConfigProvider, Layout, Row, Space, Spin, Table, Tag, message } from 'antd';
import { ImportOutlined, PoweroffOutlined, ReloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import AppSidebar from '@/layouts/AppSidebar';
import { useTheme } from '@/hooks/useTheme';
import { useStatusQuery } from '@/api/queries/useStatusQuery';
import { HttpUtil, SizeFormatter } from '@/utils';
import '@/pages/index/IndexPage.css';

interface AwgDiscovered {
  name: string;
  configPath: string;
  listenPort: number;
  address: string;
  peerCount: number;
  running: boolean;
  imported: boolean;
  inboundId?: number;
  inboundRemark?: string;
}

interface AwgInboundRuntime {
  inboundId: number;
  remark: string;
  tag: string;
  port: number;
  enable: boolean;
  interfaceName: string;
  running: boolean;
  peerCount: number;
  onlineCount: number;
  peers: Array<{
    email?: string;
    publicKey: string;
    endpoint?: string;
    online?: boolean;
    transferRx?: number;
    transferTx?: number;
  }>;
}

export default function AwgPage() {
  const { isDark, isUltra, antdThemeConfig } = useTheme();
  const navigate = useNavigate();
  const { status, refresh: refreshStatus } = useStatusQuery();
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [discovered, setDiscovered] = useState<AwgDiscovered[]>([]);
  const [inbounds, setInbounds] = useState<AwgInboundRuntime[]>([]);
  const [messageApi, messageContextHolder] = message.useMessage();

  const pageClass = `index-page ${isDark ? 'is-dark' : ''} ${isUltra ? 'is-ultra' : ''}`.trim();

  async function load() {
    setLoading(true);
    try {
      const [discoveredMsg, inboundsMsg] = await Promise.all([
        HttpUtil.get('/panel/api/awg/discovered', undefined, { silent: true }),
        HttpUtil.get('/panel/api/awg/inbounds', undefined, { silent: true }),
      ]);
      const discoveredItems = Array.isArray(discoveredMsg?.obj) ? discoveredMsg.obj as AwgDiscovered[] : [];
      const inboundItems = Array.isArray(inboundsMsg?.obj) ? inboundsMsg.obj as AwgInboundRuntime[] : [];
      setDiscovered(discoveredItems);
      setInbounds(inboundItems);
      await refreshStatus();
      return { discovered: discoveredItems, inbounds: inboundItems };
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void (async () => {
      const snapshot = await load();
      const pending = (snapshot?.discovered || []).filter((item) => !item.imported);
      if (pending.length > 0) {
        const result = await importDiscovered(false, true);
        if (result && Array.isArray(result.errors) && result.errors.length > 0) {
          messageApi.warning(result.errors.join('; '));
        }
      }
    })();
  }, []);

  const pendingImport = useMemo(
    () => discovered.filter((item) => !item.imported),
    [discovered],
  );

  async function importDiscovered(force = false, silent = false, names?: string[]) {
    setBusy(true);
    try {
      const msg = await HttpUtil.post('/panel/api/awg/scan/import', { force, names }, { silent });
      if (msg?.success) {
        const result = (msg.obj as { imported?: number; skipped?: number; errors?: string[] } | undefined) || {};
        const imported = Number(result.imported || 0);
        const errors = Array.isArray(result.errors) ? result.errors : [];
        if (!silent) {
          if (errors.length > 0) {
            messageApi.warning(errors.join('; '));
          } else if (imported > 0) {
            messageApi.success(`Imported ${imported} interface(s)`);
          } else {
            messageApi.info('No new interfaces to import');
          }
        }
        await load();
        return result;
      }
    } finally {
      setBusy(false);
    }
    return null;
  }

  async function toggleAll(enable: boolean) {
    setBusy(true);
    try {
      const msg = await HttpUtil.post('/panel/api/awg/toggle', { enable });
      if (msg?.success) {
        messageApi.success(enable ? 'AmneziaWG started' : 'AmneziaWG stopped');
        await load();
      }
    } finally {
      setBusy(false);
    }
  }

  async function restoreAll() {
    setBusy(true);
    try {
      const msg = await HttpUtil.post('/panel/api/awg/restore');
      if (msg?.success) {
        messageApi.success('Interfaces restored');
        await load();
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      <Layout className={pageClass}>
        <AppSidebar />
        <Layout className="content-shell">
          <Layout.Content className="content-area">
            <Spin spinning={loading || busy}>
              <Row gutter={[16, 16]}>
                <Col span={24}>
                  <Card
                    title="AmneziaWG"
                    extra={(
                      <Space wrap>
                        <Tag color={status.awg.installed ? 'green' : 'red'}>
                          {status.awg.installed ? `awg ${status.awg.version}` : 'awg not installed'}
                        </Tag>
                        <Badge status="processing" text={status.awg.running ? 'running' : 'stopped'} color={status.awg.running ? 'green' : 'orange'} />
                      </Space>
                    )}
                    actions={[
                      <Space className="action" key="reload" role="button" tabIndex={0} onClick={load}>
                        <ReloadOutlined />
                        <span>Reload</span>
                      </Space>,
                      <Space className="action" key="import" role="button" tabIndex={0} onClick={() => importDiscovered(pendingImport.length === 0)}>
                        <ImportOutlined />
                        <span>Import discovered</span>
                      </Space>,
                      <Space className="action" key="restore" role="button" tabIndex={0} onClick={restoreAll}>
                        <ReloadOutlined />
                        <span>Restore</span>
                      </Space>,
                      <Space className="action" key="toggle" role="button" tabIndex={0} onClick={() => toggleAll(!status.awg.running)}>
                        <PoweroffOutlined />
                        <span>{status.awg.running ? 'Stop all' : 'Start all'}</span>
                      </Space>,
                    ]}
                  >
                    {!status.awg.installed && (
                      <Alert
                        type="warning"
                        showIcon
                        message="AmneziaWG tools are not installed on this host"
                        description="Install awg and awg-quick to manage interfaces from the panel."
                        style={{ marginBottom: 16 }}
                      />
                    )}
                    <p style={{ marginBottom: 16 }}>
                      Each AmneziaWG inbound maps to one kernel interface and its config file.
                      Set
                      {' '}
                      <code>XUI_AWG_CONFIG_DIR</code>
                      {' '}
                      if Amnezia stores configs outside the default paths. Peers are managed as inbound clients.
                    </p>
                  </Card>
                </Col>

                <Col span={24}>
                  <Card title="Managed inbounds">
                    <Table
                      rowKey="inboundId"
                      dataSource={inbounds}
                      pagination={false}
                      columns={[
                        { title: 'Remark', dataIndex: 'remark' },
                        { title: 'Interface', dataIndex: 'interfaceName' },
                        { title: 'Port', dataIndex: 'port', width: 90 },
                        {
                          title: 'Runtime',
                          render: (_, row) => (
                            <Space>
                              <Tag color={row.running ? 'green' : 'default'}>{row.running ? 'up' : 'down'}</Tag>
                              <Tag color={row.enable ? 'blue' : 'default'}>{row.enable ? 'enabled' : 'disabled'}</Tag>
                            </Space>
                          ),
                        },
                        {
                          title: 'Peers',
                          render: (_, row) => `${row.onlineCount}/${row.peerCount}`,
                        },
                        {
                          title: 'Actions',
                          render: (_, row) => (
                            <Button type="link" onClick={() => navigate(`/inbounds?highlight=${row.inboundId}`)}>
                              Open inbound
                            </Button>
                          ),
                        },
                      ]}
                      expandable={{
                        expandedRowRender: (row) => (
                          <Table
                            size="small"
                            rowKey={(peer) => peer.publicKey}
                            pagination={false}
                            dataSource={row.peers}
                            columns={[
                              { title: 'Client', render: (_, peer) => peer.email || peer.publicKey },
                              { title: 'Endpoint', dataIndex: 'endpoint' },
                              {
                                title: 'Traffic',
                                render: (_, peer) => `↑ ${SizeFormatter.sizeFormat(peer.transferTx || 0)} / ↓ ${SizeFormatter.sizeFormat(peer.transferRx || 0)}`,
                              },
                              {
                                title: 'Status',
                                render: (_, peer) => <Tag color={peer.online ? 'green' : 'default'}>{peer.online ? 'online' : 'idle'}</Tag>,
                              },
                            ]}
                          />
                        ),
                      }}
                    />
                  </Card>
                </Col>

                <Col span={24}>
                  <Card title="Discovered interfaces">
                    <Table
                      rowKey="name"
                      dataSource={discovered}
                      pagination={false}
                      columns={[
                        { title: 'Interface', dataIndex: 'name' },
                        { title: 'Config', dataIndex: 'configPath' },
                        { title: 'Address', dataIndex: 'address' },
                        { title: 'Port', dataIndex: 'listenPort', width: 90 },
                        { title: 'Peers', dataIndex: 'peerCount', width: 90 },
                        {
                          title: 'State',
                          render: (_, row) => (
                            <Space>
                              <Tag color={row.running ? 'green' : 'default'}>{row.running ? 'running' : 'stopped'}</Tag>
                              <Tag color={row.imported ? 'blue' : 'orange'}>{row.imported ? 'imported' : 'new'}</Tag>
                            </Space>
                          ),
                        },
                        {
                          title: 'Inbound',
                          render: (_, row) => row.inboundRemark || '-',
                        },
                        {
                          title: 'Actions',
                          width: 120,
                          render: (_, row) => (
                            row.imported ? (
                              <Button type="link" onClick={() => navigate(`/inbounds?highlight=${row.inboundId}`)}>
                                Open
                              </Button>
                            ) : (
                              <Button
                                type="link"
                                loading={busy}
                                onClick={() => void importDiscovered(false, false, [row.name])}
                              >
                                Import
                              </Button>
                            )
                          ),
                        },
                      ]}
                    />
                  </Card>
                </Col>
              </Row>
            </Spin>
          </Layout.Content>
        </Layout>
      </Layout>
    </ConfigProvider>
  );
}

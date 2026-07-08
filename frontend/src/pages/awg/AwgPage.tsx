import { useEffect, useMemo, useState } from 'react';
import { Alert, Badge, Button, Card, Col, ConfigProvider, Form, Input, InputNumber, Layout, Modal, Popconfirm, Row, Space, Spin, Switch, Tag, message } from 'antd';
import { CopyOutlined, DeleteOutlined, EditOutlined, KeyOutlined, PoweroffOutlined, ReloadOutlined, SaveOutlined } from '@ant-design/icons';

import AppSidebar from '@/layouts/AppSidebar';
import { useTheme } from '@/hooks/useTheme';
import { useStatusQuery } from '@/api/queries/useStatusQuery';
import { ClipboardManager, HttpUtil, SizeFormatter, Wireguard } from '@/utils';
import type { AwgPeer } from '@/models/status';
import '@/pages/index/IndexPage.css';

interface AwgFormValues {
  id: number;
  enable: boolean;
  interfaceName: string;
  listenPort: number;
  ipv4Address: string;
  ipv4Pool: string;
  externalInterface: string;
  privateKey: string;
  publicKey: string;
  endpoint: string;
  dns: string;
  mtu: number;
  jc: number;
  jmin: number;
  jmax: number;
  s1: number;
  s2: number;
  s3: number;
  s4: number;
  h1: string;
  h2: string;
  h3: string;
  h4: string;
  i1: string;
  i2: string;
  i3: string;
  i4: string;
  i5: string;
  postUp: string;
  postDown: string;
  trafficReset: string;
}

interface AwgClientRow {
  id: number;
  uuid: string;
  name: string;
  email: string;
  enable: boolean;
  comment: string;
  publicKey: string;
  privateKey: string;
  jc: number;
  jmin: number;
  jmax: number;
  i1: string;
  i2: string;
  i3: string;
  i4: string;
  i5: string;
  allowedIPs: string;
  clientAllowedIPs: string;
  persistentKeepalive: number;
  upload: number;
  download: number;
  lastOnline: number;
  lastIp: string;
}

interface AwgClientFormValues {
  id: number;
  name: string;
  email: string;
  enable: boolean;
  comment: string;
  allowedIPs: string;
  clientAllowedIPs: string;
  persistentKeepalive: number;
  jc: number;
  jmin: number;
  jmax: number;
  i1: string;
  i2: string;
  i3: string;
  i4: string;
  i5: string;
}

const defaultForm: AwgFormValues = {
  id: 0,
  enable: false,
  interfaceName: 'awg0',
  listenPort: 51820,
  ipv4Address: '10.66.66.1/24',
  ipv4Pool: '10.66.66.0/24',
  externalInterface: '',
  privateKey: '',
  publicKey: '',
  endpoint: '',
  dns: '1.1.1.1,2606:4700:4700::1111',
  mtu: 1420,
  jc: 4,
  jmin: 64,
  jmax: 256,
  s1: 15,
  s2: 25,
  s3: 35,
  s4: 15,
  h1: '1',
  h2: '2',
  h3: '3',
  h4: '4',
  i1: '',
  i2: '',
  i3: '',
  i4: '',
  i5: '',
  postUp: '',
  postDown: '',
  trafficReset: 'never',
};

function formFromServer(server: Partial<AwgFormValues> | null): AwgFormValues {
  if (!server) {
    const kp = Wireguard.generateKeypair();
    return { ...defaultForm, privateKey: kp.privateKey, publicKey: kp.publicKey };
  }
  const privateKey = String(server.privateKey || '');
  return {
    ...defaultForm,
    ...server,
    id: Number(server.id || 0),
    enable: server.enable === true,
    listenPort: Number(server.listenPort || defaultForm.listenPort),
    mtu: Number(server.mtu || defaultForm.mtu),
    privateKey,
    publicKey: privateKey ? Wireguard.generateKeypair(privateKey).publicKey : String(server.publicKey || ''),
  };
}

function payloadFromForm(values: AwgFormValues) {
  return values;
}

function formatHandshake(peer: AwgPeer) {
  if (!peer.latestHandshake) return 'no handshake';
  return new Date(peer.latestHandshake * 1000).toLocaleString();
}

export default function AwgPage() {
  const { isDark, isUltra, antdThemeConfig } = useTheme();
  const { status, refresh: refreshStatus } = useStatusQuery();
  const [form] = Form.useForm<AwgFormValues>();
  const [clientForm] = Form.useForm<AwgClientFormValues>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [runtimeBusy, setRuntimeBusy] = useState(false);
  const [clients, setClients] = useState<AwgClientRow[]>([]);
  const [editingClient, setEditingClient] = useState<AwgClientRow | null>(null);
  const [clientBusy, setClientBusy] = useState(false);
  const [messageApi, messageContextHolder] = message.useMessage();

  const pageClass = `index-page ${isDark ? 'is-dark' : ''} ${isUltra ? 'is-ultra' : ''}`.trim();

  async function load() {
    setLoading(true);
    try {
      const msg = await HttpUtil.get('/panel/api/awg/server', undefined, { silent: true });
      form.setFieldsValue(formFromServer((msg?.obj || null) as Partial<AwgFormValues> | null));
      const clientsMsg = await HttpUtil.get('/panel/api/awg/clients', undefined, { silent: true });
      setClients(Array.isArray(clientsMsg?.obj) ? clientsMsg.obj as AwgClientRow[] : []);
      await refreshStatus();
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const headerExtra = useMemo(() => (
    <Space wrap>
      <Tag color={status.awg.installed ? 'green' : 'red'}>
        {status.awg.installed ? `awg ${status.awg.version}` : 'awg not installed'}
      </Tag>
      <Tag color={status.awg.running ? 'green' : 'orange'}>
        {status.awg.running ? 'running' : 'stopped'}
      </Tag>
      <Tag color={status.awg.onlineCount > 0 ? 'green' : 'default'}>
        {status.awg.onlineCount}/{status.awg.peerCount} peers
      </Tag>
    </Space>
  ), [status.awg.installed, status.awg.onlineCount, status.awg.peerCount, status.awg.running, status.awg.version]);

  async function save() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const payload = payloadFromForm(values);
      const msg = await HttpUtil.post('/panel/api/awg/server', payload);
      if (msg?.success) {
        messageApi.success('Saved');
        await load();
      }
    } finally {
      setSaving(false);
    }
  }

  function regenerateServerKey() {
    const kp = Wireguard.generateKeypair();
    form.setFieldsValue({ privateKey: kp.privateKey, publicKey: kp.publicKey });
  }

  async function setAwgEnabled(enable: boolean) {
    setRuntimeBusy(true);
    try {
      const msg = await HttpUtil.post('/panel/api/awg/server/toggle', { enable });
      if (msg?.success) {
        messageApi.success(enable ? 'AmneziaWG started' : 'AmneziaWG stopped');
        await load();
      }
    } finally {
      setRuntimeBusy(false);
    }
  }

  async function restartAwg() {
    setRuntimeBusy(true);
    try {
      await HttpUtil.post('/panel/api/awg/server/toggle', { enable: false }, { silent: true });
      const msg = await HttpUtil.post('/panel/api/awg/server/toggle', { enable: true });
      if (msg?.success) {
        messageApi.success('AmneziaWG restarted');
        await load();
      }
    } finally {
      setRuntimeBusy(false);
    }
  }

  function editClient(client: AwgClientRow) {
    setEditingClient(client);
    clientForm.setFieldsValue({
      id: client.id,
      name: client.name,
      email: client.email,
      enable: client.enable,
      comment: client.comment,
      allowedIPs: client.allowedIPs,
      clientAllowedIPs: client.clientAllowedIPs || '0.0.0.0/0, ::/0',
      persistentKeepalive: client.persistentKeepalive || 25,
      jc: client.jc || 4,
      jmin: client.jmin || 64,
      jmax: client.jmax || 256,
      i1: client.i1 || '',
      i2: client.i2 || '',
      i3: client.i3 || '',
      i4: client.i4 || '',
      i5: client.i5 || '',
    });
  }

  async function saveClient() {
    const values = await clientForm.validateFields();
    setClientBusy(true);
    try {
      const msg = await HttpUtil.post(`/panel/api/awg/client/update/${values.id}`, values);
      if (msg?.success) {
        messageApi.success('Client saved');
        setEditingClient(null);
        await load();
      }
    } finally {
      setClientBusy(false);
    }
  }

  async function toggleClient(client: AwgClientRow) {
    setClientBusy(true);
    try {
      const msg = await HttpUtil.post(`/panel/api/awg/client/toggle/${client.id}`, { enable: !client.enable });
      if (msg?.success) {
        messageApi.success(!client.enable ? 'Client enabled' : 'Client disabled');
        await load();
      }
    } finally {
      setClientBusy(false);
    }
  }

  async function reissueClient(client: AwgClientRow) {
    setClientBusy(true);
    try {
      const msg = await HttpUtil.post(`/panel/api/awg/client/reissue/${client.id}`);
      if (msg?.success) {
        messageApi.success('Client reissued');
        await load();
      }
    } finally {
      setClientBusy(false);
    }
  }

  async function deleteClient(client: AwgClientRow) {
    setClientBusy(true);
    try {
      const msg = await HttpUtil.post(`/panel/api/awg/client/del/${client.id}`);
      if (msg?.success) {
        messageApi.success('Client deleted');
        await load();
      }
    } finally {
      setClientBusy(false);
    }
  }

  async function copyClientConfig(client: AwgClientRow) {
    const msg = await HttpUtil.get(`/panel/api/awg/client/${client.id}/config`, undefined, { silent: true });
    if (msg?.success && typeof msg.obj === 'string') {
      const ok = await ClipboardManager.copyText(msg.obj);
      if (ok) messageApi.success('Config copied');
      return;
    }
    messageApi.warning('Reissue this runtime import before copying config');
  }

  const peerRows = status.awg.peers || [];

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      <Layout className={pageClass}>
        <AppSidebar />
        <Layout className="content-shell">
          <Layout.Content className="content-area">
            <Spin spinning={loading}>
              <Row gutter={[16, 16]}>
                <Col span={24}>
                  <Card
                    title="AmneziaWG runtime"
                    extra={headerExtra}
                    actions={[
                      <Space className="action" key="reload" role="button" tabIndex={0} onClick={load}>
                        <ReloadOutlined />
                        <span>Reload</span>
                      </Space>,
                      <Space className="action" key="toggle" role="button" tabIndex={0} onClick={() => setAwgEnabled(!status.awg.running)}>
                        <PoweroffOutlined />
                        <span>{status.awg.running ? 'Stop' : 'Start'}</span>
                      </Space>,
                      <Space className="action" key="restart" role="button" tabIndex={0} onClick={restartAwg}>
                        <ReloadOutlined />
                        <span>Restart</span>
                      </Space>,
                    ]}
                  >
                    <Spin spinning={runtimeBusy}>
                      {status.awg.error && <Alert type="error" showIcon message={status.awg.error} style={{ marginBottom: 12 }} />}
                      {!status.awg.installed && <Alert type="warning" showIcon message="awg runtime is not installed on this server" style={{ marginBottom: 12 }} />}
                      <div className="awg-peer-list">
                        {peerRows.length === 0 ? (
                          <Tag>no runtime peers</Tag>
                        ) : peerRows.map((peer, idx) => (
                          <div className="awg-peer-row" key={`${peer.publicKey || idx}`}>
                            <div className="awg-peer-main">
                              <span>{peer.email || peer.publicKey || `peer-${idx + 1}`}</span>
                              <Badge status={peer.online ? 'success' : 'default'} text={peer.online ? 'online' : 'idle'} />
                            </div>
                            <div className="awg-peer-meta">{peer.interfaceName || 'awg'} / {peer.endpoint || 'no endpoint'}</div>
                            <div className="awg-peer-meta">{(peer.allowedIPs || []).join(', ') || 'no allowed IPs'}</div>
                            <div className="awg-peer-meta">
                              up {SizeFormatter.sizeFormat(peer.transferTx || 0)} / down {SizeFormatter.sizeFormat(peer.transferRx || 0)}
                            </div>
                            <div className="awg-peer-meta">{formatHandshake(peer)}</div>
                          </div>
                        ))}
                      </div>
                    </Spin>
                  </Card>
                </Col>

                <Col span={24}>
                  <Card title="AmneziaWGv2 clients" extra={<Tag>{clients.length} clients</Tag>}>
                    <div className="awg-peer-list">
                      {clients.length === 0 ? (
                        <Tag>no clients</Tag>
                      ) : clients.map((client) => (
                        <div className="awg-peer-row" key={client.uuid || client.id}>
                          <div className="awg-peer-main">
                            <span>{client.name || client.email || client.uuid}</span>
                            <Badge status={client.enable ? 'success' : 'default'} text={client.enable ? 'enabled' : 'disabled'} />
                            {!client.privateKey && <Tag color="orange">runtime import</Tag>}
                            <Space size={4} wrap>
                              <Button size="small" icon={<EditOutlined />} onClick={() => editClient(client)}>Edit</Button>
                              <Button size="small" icon={<PoweroffOutlined />} onClick={() => toggleClient(client)}>
                                {client.enable ? 'Disable' : 'Enable'}
                              </Button>
                              <Button size="small" icon={<KeyOutlined />} onClick={() => reissueClient(client)}>Reissue</Button>
                              <Button size="small" icon={<CopyOutlined />} onClick={() => copyClientConfig(client)}>Config</Button>
                              <Popconfirm title="Delete AWG client?" onConfirm={() => deleteClient(client)}>
                                <Button size="small" danger icon={<DeleteOutlined />}>Delete</Button>
                              </Popconfirm>
                            </Space>
                          </div>
                          <div className="awg-peer-meta">{client.email || 'no email'} / {client.lastIp || 'no endpoint'}</div>
                          <div className="awg-peer-meta">{client.allowedIPs || 'no allowed IPs'}</div>
                          <div className="awg-peer-meta">{client.publicKey || 'no public key'}</div>
                          <div className="awg-peer-meta">
                            up {SizeFormatter.sizeFormat(client.upload || 0)} / down {SizeFormatter.sizeFormat(client.download || 0)}
                          </div>
                        </div>
                      ))}
                    </div>
                  </Card>
                </Col>

                <Col span={24}>
                  <Card
                    title="AmneziaWG settings"
                    extra={<Tag>AmneziaWGv2</Tag>}
                    actions={[
                      <Space className="action" key="reload" role="button" tabIndex={0} onClick={load}>
                        <ReloadOutlined />
                        <span>Reload</span>
                      </Space>,
                      <Space className="action" key="save" role="button" tabIndex={0} onClick={save}>
                        <SaveOutlined />
                        <span>Save</span>
                      </Space>,
                    ]}
                  >
                    <Form form={form} layout="vertical" initialValues={defaultForm} disabled={saving}>
                      <Form.Item name="id" hidden><InputNumber /></Form.Item>
                      <Row gutter={16}>
                        <Col xs={24} md={8}>
                          <Form.Item name="enable" label="Enabled" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={8}>
                          <Form.Item name="interfaceName" label="Interface" rules={[{ required: true }]}>
                            <Input />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={8}>
                          <Form.Item name="listenPort" label="Listen port" rules={[{ required: true }]}>
                            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={8}>
                          <Form.Item name="ipv4Address" label="Server IPv4" rules={[{ required: true }]}>
                            <Input placeholder="10.66.66.1/24" />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={8}>
                          <Form.Item name="ipv4Pool" label="IPv4 pool" rules={[{ required: true }]}>
                            <Input placeholder="10.66.66.0/24" />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={8}>
                          <Form.Item name="externalInterface" label="External interface">
                            <Input placeholder="eth0" />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={8}>
                          <Form.Item name="endpoint" label="Endpoint">
                            <Input placeholder="example.com:51820" />
                          </Form.Item>
                        </Col>
                        <Col xs={24}>
                          <Form.Item label="Server private key" required>
                            <Space.Compact style={{ display: 'flex' }}>
                              <Form.Item name="privateKey" noStyle rules={[{ required: true }]}>
                                <Input
                                  style={{ flex: 1 }}
                                  onChange={(event) => {
                                    const privateKey = event.target.value;
                                    form.setFieldValue('publicKey', privateKey ? Wireguard.generateKeypair(privateKey).publicKey : '');
                                  }}
                                />
                              </Form.Item>
                              <Button icon={<ReloadOutlined />} onClick={regenerateServerKey} />
                            </Space.Compact>
                          </Form.Item>
                        </Col>
                        <Col xs={24}>
                          <Form.Item name="publicKey" label="Server public key">
                            <Input disabled />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item name="dns" label="Client DNS">
                            <Input />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item name="mtu" label="MTU">
                            <InputNumber min={1} style={{ width: '100%' }} />
                          </Form.Item>
                        </Col>
                      </Row>

                      <Row gutter={12}>
                        {(['jc', 'jmin', 'jmax', 's1', 's2', 's3', 's4'] as const).map((key) => (
                          <Col xs={12} md={8} lg={4} key={key}>
                            <Form.Item name={key} label={key.toUpperCase()}>
                              <InputNumber min={0} style={{ width: '100%' }} />
                            </Form.Item>
                          </Col>
                        ))}
                      </Row>

                      <Row gutter={16}>
                        {(['h1', 'h2', 'h3', 'h4'] as const).map((key) => (
                          <Col xs={24} md={12} lg={6} key={key}>
                            <Form.Item name={key} label={key.toUpperCase()}>
                              <Input />
                            </Form.Item>
                          </Col>
                        ))}
                      </Row>

                      <Row gutter={16}>
                        {(['i1', 'i2', 'i3', 'i4', 'i5'] as const).map((key) => (
                          <Col xs={24} md={12} lg={8} key={key}>
                            <Form.Item name={key} label={key.toUpperCase()}>
                              <Input />
                            </Form.Item>
                          </Col>
                        ))}
                      </Row>

                      <Row gutter={16}>
                        <Col xs={24} md={12}>
                          <Form.Item name="postUp" label="PostUp">
                            <Input.TextArea autoSize={{ minRows: 3, maxRows: 6 }} />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item name="postDown" label="PostDown">
                            <Input.TextArea autoSize={{ minRows: 3, maxRows: 6 }} />
                          </Form.Item>
                        </Col>
                      </Row>

                      <Space>
                        <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={save}>Save</Button>
                        <Button icon={<ReloadOutlined />} onClick={load}>Reload</Button>
                      </Space>
                    </Form>
                  </Card>
                </Col>
              </Row>
            </Spin>
          </Layout.Content>
        </Layout>
      </Layout>
      <Modal
        open={!!editingClient}
        title="Edit AmneziaWGv2 client"
        okText="Save"
        confirmLoading={clientBusy}
        onOk={saveClient}
        onCancel={() => setEditingClient(null)}
      >
        <Form form={clientForm} layout="vertical">
          <Form.Item name="id" hidden><InputNumber /></Form.Item>
          <Form.Item name="enable" label="Enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="name" label="Name">
            <Input />
          </Form.Item>
          <Form.Item name="email" label="Email" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="comment" label="Comment">
            <Input />
          </Form.Item>
          <Form.Item name="allowedIPs" label="Peer allowed IPs" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="clientAllowedIPs" label="Client route allowed IPs">
            <Input />
          </Form.Item>
          <Form.Item name="persistentKeepalive" label="Persistent keepalive">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Row gutter={12}>
            {(['jc', 'jmin', 'jmax'] as const).map((key) => (
              <Col span={8} key={key}>
                <Form.Item name={key} label={key.toUpperCase()}>
                  <InputNumber min={0} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            ))}
          </Row>
          <Row gutter={12}>
            {(['i1', 'i2', 'i3', 'i4', 'i5'] as const).map((key) => (
              <Col xs={24} md={12} key={key}>
                <Form.Item name={key} label={key.toUpperCase()}>
                  <Input />
                </Form.Item>
              </Col>
            ))}
          </Row>
        </Form>
      </Modal>
    </ConfigProvider>
  );
}

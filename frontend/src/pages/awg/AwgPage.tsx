import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Col, ConfigProvider, Form, Input, InputNumber, Layout, Row, Space, Spin, Switch, Tag, message } from 'antd';
import { ReloadOutlined, SaveOutlined } from '@ant-design/icons';

import AppSidebar from '@/layouts/AppSidebar';
import { useTheme } from '@/hooks/useTheme';
import { useStatusQuery } from '@/api/queries/useStatusQuery';
import { HttpUtil, RandomUtil, Wireguard } from '@/utils';
import { createDefaultAmneziawgInboundSettings } from '@/lib/xray/inbound-defaults';
import { coerceInboundJsonField, type DBInboundInit } from '@/models/dbinbound';
import '@/pages/index/IndexPage.css';

interface AwgFormValues {
  id: number;
  enable: boolean;
  remark: string;
  listen: string;
  port: number;
  address: string;
  externalInterface: string;
  secretKey: string;
  publicKey: string;
  dns: string;
  mtu: number;
  jc: number;
  jmin: number;
  jmax: number;
  s1: number;
  s2: number;
  h1: number;
  h2: number;
  h3: number;
  h4: number;
  postUp: string;
  postDown: string;
}

const defaultForm: AwgFormValues = {
  id: 0,
  enable: false,
  remark: 'AmneziaWG',
  listen: '',
  port: 51820,
  address: '10.66.66.1/24',
  externalInterface: '',
  secretKey: '',
  publicKey: '',
  dns: '1.1.1.1,2606:4700:4700::1111',
  mtu: 1420,
  jc: 4,
  jmin: 50,
  jmax: 1000,
  s1: 0,
  s2: 0,
  h1: 1,
  h2: 2,
  h3: 3,
  h4: 4,
  postUp: '',
  postDown: '',
};

function formFromInbound(inbound: DBInboundInit | null): AwgFormValues {
  if (!inbound) {
    const kp = Wireguard.generateKeypair();
    return { ...defaultForm, secretKey: kp.privateKey, publicKey: kp.publicKey };
  }
  const settings = coerceInboundJsonField(inbound.settings);
  const secretKey = String(settings.secretKey || '');
  return {
    ...defaultForm,
    id: inbound.id || 0,
    enable: inbound.enable !== false,
    remark: inbound.remark || defaultForm.remark,
    listen: inbound.listen || '',
    port: Number(inbound.port || defaultForm.port),
    address: String(settings.address || defaultForm.address),
    externalInterface: String(settings.externalInterface || ''),
    secretKey,
    publicKey: secretKey ? Wireguard.generateKeypair(secretKey).publicKey : '',
    dns: String(settings.dns || defaultForm.dns),
    mtu: Number(settings.mtu || defaultForm.mtu),
    jc: Number(settings.jc ?? defaultForm.jc),
    jmin: Number(settings.jmin ?? defaultForm.jmin),
    jmax: Number(settings.jmax ?? defaultForm.jmax),
    s1: Number(settings.s1 ?? defaultForm.s1),
    s2: Number(settings.s2 ?? defaultForm.s2),
    h1: Number(settings.h1 ?? defaultForm.h1),
    h2: Number(settings.h2 ?? defaultForm.h2),
    h3: Number(settings.h3 ?? defaultForm.h3),
    h4: Number(settings.h4 ?? defaultForm.h4),
    postUp: String(settings.postUp || ''),
    postDown: String(settings.postDown || ''),
  };
}

function payloadFromForm(values: AwgFormValues) {
  const settings = {
    ...createDefaultAmneziawgInboundSettings({ secretKey: values.secretKey, mtu: values.mtu }),
    address: values.address,
    externalInterface: values.externalInterface,
    dns: values.dns,
    jc: values.jc,
    jmin: values.jmin,
    jmax: values.jmax,
    s1: values.s1,
    s2: values.s2,
    h1: values.h1,
    h2: values.h2,
    h3: values.h3,
    h4: values.h4,
    postUp: values.postUp,
    postDown: values.postDown,
  };
  return {
    up: 0,
    down: 0,
    total: 0,
    remark: values.remark || defaultForm.remark,
    enable: values.enable,
    expiryTime: 0,
    listen: values.listen || '',
    port: values.port,
    protocol: 'amneziawg',
    settings: JSON.stringify(settings, null, 2),
    streamSettings: JSON.stringify({ network: 'udp', security: 'none' }),
    sniffing: JSON.stringify({ enabled: false, destOverride: [] }),
    tag: values.id ? undefined : `inbound-amneziawg-${RandomUtil.randomLowerAndNum(6)}`,
  };
}

export default function AwgPage() {
  const { isDark, isUltra, antdThemeConfig } = useTheme();
  const { status, refresh: refreshStatus } = useStatusQuery();
  const [form] = Form.useForm<AwgFormValues>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [inbound, setInbound] = useState<DBInboundInit | null>(null);
  const [messageApi, messageContextHolder] = message.useMessage();

  const pageClass = `index-page ${isDark ? 'is-dark' : ''} ${isUltra ? 'is-ultra' : ''}`.trim();

  async function load() {
    setLoading(true);
    try {
      const msg = await HttpUtil.get('/panel/api/inbounds/list', undefined, { silent: true });
      const list = Array.isArray(msg?.obj) ? (msg.obj as DBInboundInit[]) : [];
      const found = list.find((row) => row.protocol === 'amneziawg') || null;
      setInbound(found);
      form.setFieldsValue(formFromInbound(found));
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
    </Space>
  ), [status.awg.installed, status.awg.running, status.awg.version]);

  async function save() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const payload = payloadFromForm(values);
      const url = values.id ? `/panel/api/inbounds/update/${values.id}` : '/panel/api/inbounds/add';
      const msg = await HttpUtil.post(url, payload);
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
    form.setFieldsValue({ secretKey: kp.privateKey, publicKey: kp.publicKey });
  }

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      <Layout className={pageClass}>
        <AppSidebar />
        <Layout className="content-shell">
          <Layout.Content className="content-area">
            <Spin spinning={loading}>
              <Card
                title="AmneziaWG"
                extra={headerExtra}
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
                      <Form.Item name="remark" label="Name" rules={[{ required: true }]}>
                        <Input />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name="port" label="Listen port" rules={[{ required: true }]}>
                        <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name="listen" label="Listen IP">
                        <Input placeholder="0.0.0.0" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name="address" label="Server address" rules={[{ required: true }]}>
                        <Input placeholder="10.66.66.1/24" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={8}>
                      <Form.Item name="externalInterface" label="External interface">
                        <Input placeholder="eth0" />
                      </Form.Item>
                    </Col>
                    <Col xs={24}>
                      <Form.Item label="Server private key" required>
                        <Space.Compact style={{ display: 'flex' }}>
                          <Form.Item name="secretKey" noStyle rules={[{ required: true }]}>
                            <Input
                              style={{ flex: 1 }}
                              onChange={(event) => {
                                const secretKey = event.target.value;
                                form.setFieldValue('publicKey', secretKey ? Wireguard.generateKeypair(secretKey).publicKey : '');
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
                    {(['jc', 'jmin', 'jmax', 's1', 's2', 'h1', 'h2', 'h3', 'h4'] as const).map((key) => (
                      <Col xs={12} md={8} lg={4} key={key}>
                        <Form.Item name={key} label={key.toUpperCase()}>
                          <InputNumber min={0} style={{ width: '100%' }} />
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
                    {inbound?.id && <Tag>inbound #{inbound.id}</Tag>}
                  </Space>
                </Form>
              </Card>
            </Spin>
          </Layout.Content>
        </Layout>
      </Layout>
    </ConfigProvider>
  );
}

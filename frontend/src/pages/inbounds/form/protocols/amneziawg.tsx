import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Col, Collapse, Descriptions, Form, Input, InputNumber, Row, Select, Space, message } from 'antd';
import { ImportOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { HttpUtil } from '@/utils';

interface AwgDiscovered {
  name: string;
  listenPort: number;
  running: boolean;
  imported: boolean;
}

interface AwgInboundTemplate {
  remark: string;
  port: number;
  enable: boolean;
  tag: string;
  settings: Record<string, unknown>;
}

interface AwgProvisionResult {
  remark: string;
  port: number;
  enable: boolean;
  tag: string;
  publicKey: string;
  interfaceName: string;
  configPath: string;
  settings: Record<string, unknown>;
}

interface AmneziawgFieldsProps {
  wgPubKey: string;
  regenInboundWg: () => void;
  mode: 'create' | 'edit';
}

export default function AmneziawgFields({ wgPubKey, regenInboundWg, mode }: AmneziawgFieldsProps) {
  const { t } = useTranslation();
  const form = Form.useFormInstance();
  const [messageApi, messageContextHolder] = message.useMessage();
  const [discovered, setDiscovered] = useState<AwgDiscovered[]>([]);
  const [loading, setLoading] = useState(false);
  const [adoptName, setAdoptName] = useState<string>();
  const [provision, setProvision] = useState<AwgProvisionResult | null>(null);
  const port = Form.useWatch('port', form) as number | undefined;
  const awgInterface = Form.useWatch(['settings', 'awgInterface'], form) as string | undefined;
  const serverAddress = Form.useWatch(['settings', 'address'], form) as string | undefined;

  const pendingAdopt = useMemo(
    () => discovered.filter((item) => !item.imported),
    [discovered],
  );

  const applyProvision = useCallback((next: AwgProvisionResult, keepRemark = true) => {
    const currentRemark = form.getFieldValue('remark') as string | undefined;
    form.setFieldsValue({
      remark: keepRemark && currentRemark?.trim() ? currentRemark : next.remark,
      port: next.port,
      enable: next.enable,
      tag: next.tag,
      settings: next.settings,
    });
    setProvision(next);
    setAdoptName(undefined);
  }, [form]);

  const loadProvision = useCallback(async (keepRemark = true) => {
    setLoading(true);
    try {
      const msg = await HttpUtil.get('/panel/api/awg/provision/new', undefined, { silent: true });
      if (!msg?.success || !msg.obj) {
        messageApi.error('Unable to prepare AWGv2 interface');
        return;
      }
      applyProvision(msg.obj as AwgProvisionResult, keepRemark);
    } finally {
      setLoading(false);
    }
  }, [applyProvision, messageApi]);

  useEffect(() => {
    if (mode !== 'create') return;
    void loadProvision(false);
    void HttpUtil.get('/panel/api/awg/discovered', undefined, { silent: true })
      .then((msg) => {
        setDiscovered(Array.isArray(msg?.obj) ? msg.obj as AwgDiscovered[] : []);
      });
  }, [loadProvision, mode]);

  useEffect(() => {
    if (mode !== 'create' || adoptName) return;
    const iface = typeof awgInterface === 'string' ? awgInterface.trim() : '';
    if (iface === 'awg0') return;
    const nextPort = typeof port === 'number' ? port : 0;
    if (nextPort > 0) {
      form.setFieldValue(['settings', 'awgInterface'], `awg_in_${nextPort}_ud`);
    }
  }, [adoptName, awgInterface, form, mode, port]);

  async function applyTemplate(name: string) {
    setLoading(true);
    try {
      const msg = await HttpUtil.get(`/panel/api/awg/discovered/${encodeURIComponent(name)}/template`);
      if (!msg?.success || !msg.obj) return;
      const template = msg.obj as AwgInboundTemplate;
      form.setFieldsValue({
        remark: template.remark,
        port: template.port,
        enable: template.enable,
        tag: template.tag,
        settings: template.settings,
      });
      setAdoptName(name);
      setProvision(null);
      messageApi.success(`Loaded ${name}`);
    } finally {
      setLoading(false);
    }
  }

  async function importSelected() {
    if (!adoptName) return;
    const msg = await HttpUtil.post('/panel/api/awg/scan/import', { force: false, names: [adoptName] });
    if (!msg?.success) return;
    const result = (msg.obj as { imported?: number; errors?: string[] } | undefined) || {};
    if (Array.isArray(result.errors) && result.errors.length > 0) {
      messageApi.warning(result.errors.join('; '));
      return;
    }
    messageApi.success(result.imported ? `Imported ${result.imported} interface(s)` : 'Interface already managed');
  }

  return (
    <>
      {messageContextHolder}
      {mode === 'create' && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="AWGv2 auto-provision"
          description={(
            <>
              The panel picks the next free Amnezia interface name, UDP port, subnet, keys, and obfuscation
              parameters. Only the inbound remark is required; everything else can be tuned below before save.
            </>
          )}
        />
      )}

      {mode === 'create' && provision && (
        <Descriptions
          bordered
          size="small"
          column={1}
          style={{ marginBottom: 16 }}
          title="Generated runtime"
        >
          <Descriptions.Item label="Interface">{provision.interfaceName}</Descriptions.Item>
          <Descriptions.Item label="UDP port">{provision.port}</Descriptions.Item>
          <Descriptions.Item label="Server address">{serverAddress || provision.settings.address as string}</Descriptions.Item>
          <Descriptions.Item label="Config file">{provision.configPath}</Descriptions.Item>
          <Descriptions.Item label="Public key">{provision.publicKey}</Descriptions.Item>
        </Descriptions>
      )}

      {mode === 'create' && (
        <Space style={{ marginBottom: 16 }}>
          <Button loading={loading} icon={<ReloadOutlined />} onClick={() => void loadProvision(true)}>
            Regenerate plan
          </Button>
        </Space>
      )}

      <Collapse
        defaultActiveKey={mode === 'edit' ? ['runtime'] : []}
        items={[
          {
            key: 'runtime',
            label: 'Runtime settings',
            children: (
              <>
                <Form.Item label={t('pages.xray.wireguard.secretKey')}>
                  <Space.Compact block>
                    <Form.Item name={['settings', 'secretKey']} noStyle>
                      <Input style={{ width: 'calc(100% - 32px)' }} />
                    </Form.Item>
                    <Button aria-label={t('regenerate')} icon={<ReloadOutlined />} onClick={regenInboundWg} />
                  </Space.Compact>
                </Form.Item>
                <Form.Item label={t('pages.xray.wireguard.publicKey')}>
                  <Input value={wgPubKey} disabled />
                </Form.Item>
                <Form.Item name={['settings', 'address']} label="Server address">
                  <Input placeholder="10.66.66.1/24" />
                </Form.Item>
                <Form.Item
                  name={['settings', 'awgInterface']}
                  label="Kernel interface"
                  extra="awg0 for the first tunnel, awg_in_{port}_ud for the next ones"
                >
                  <Input placeholder="awg0" />
                </Form.Item>
                <Form.Item name="port" label="Listen port (UDP)" hidden={mode === 'create'}>
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                </Form.Item>
                <Form.Item name={['settings', 'mtu']} label="MTU">
                  <InputNumber min={1280} style={{ width: '100%' }} />
                </Form.Item>
                <Form.Item name={['settings', 'dns']} label={t('pages.inbounds.info.dns')}>
                  <Input placeholder="1.1.1.1, 1.0.0.1" />
                </Form.Item>
                <Form.Item name={['settings', 'externalInterface']} label="External interface">
                  <Input placeholder="eth0" />
                </Form.Item>
              </>
            ),
          },
          {
            key: 'obfuscation',
            label: 'AWGv2 obfuscation',
            children: (
              <>
                <Row gutter={12}>
                  <Col xs={24} md={8}>
                    <Form.Item name={['settings', 'jc']} label="Jc">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item name={['settings', 'jmin']} label="Jmin">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item name={['settings', 'jmax']} label="Jmax">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={12}>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 's1']} label="S1">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 's2']} label="S2">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 's3']} label="S3">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 's4']} label="S4">
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={12}>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 'h1']} label="H1">
                      <Input />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 'h2']} label="H2">
                      <Input />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 'h3']} label="H3">
                      <Input />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item name={['settings', 'h4']} label="H4">
                      <Input />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={12}>
                  {(['i1', 'i2', 'i3', 'i4', 'i5'] as const).map((key) => (
                    <Col xs={24} md={12} key={key}>
                      <Form.Item name={['settings', key]} label={key.toUpperCase()}>
                        <Input />
                      </Form.Item>
                    </Col>
                  ))}
                </Row>
              </>
            ),
          },
          {
            key: 'hooks',
            label: 'PostUp / PostDown',
            children: (
              <>
                <Form.Item name={['settings', 'postUp']} label="PostUp">
                  <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
                </Form.Item>
                <Form.Item name={['settings', 'postDown']} label="PostDown">
                  <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
                </Form.Item>
              </>
            ),
          },
          ...(mode === 'create' && pendingAdopt.length > 0 ? [{
            key: 'adopt',
            label: 'Adopt existing Amnezia interface',
            children: (
              <Form.Item label="Interface on host">
                <Space.Compact block>
                  <Select
                    allowClear
                    loading={loading}
                    placeholder="Select interface already created by Amnezia"
                    style={{ width: 'calc(100% - 96px)' }}
                    value={adoptName}
                    options={pendingAdopt.map((item) => ({
                      value: item.name,
                      label: `${item.name}${item.running ? ' (running)' : ''} :${item.listenPort || '?'}`,
                    }))}
                    onChange={(value) => {
                      setAdoptName(value);
                      if (value) void applyTemplate(value);
                    }}
                  />
                  <Button icon={<ImportOutlined />} onClick={() => void importSelected()} disabled={!adoptName}>
                    Import
                  </Button>
                </Space.Compact>
              </Form.Item>
            ),
          }] : []),
        ]}
      />
    </>
  );
}

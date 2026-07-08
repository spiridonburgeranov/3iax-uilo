import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Col, Collapse, Descriptions, Form, Input, InputNumber, Row, Select, Space, message } from 'antd';
import { ImportOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { HttpUtil } from '@/utils';
import { buildLocalAwgProvision } from '@/lib/awg/provision';

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
  mode: 'add' | 'edit';
}

export default function AmneziawgFields({ wgPubKey, regenInboundWg, mode }: AmneziawgFieldsProps) {
  const { t } = useTranslation();
  const form = Form.useFormInstance();
  const isCreate = mode === 'add';
  const [messageApi, messageContextHolder] = message.useMessage();
  const [discovered, setDiscovered] = useState<AwgDiscovered[]>([]);
  const [loading, setLoading] = useState(false);
  const [adoptName, setAdoptName] = useState<string>();
  const [provision, setProvision] = useState<AwgProvisionResult | null>(null);
  const serverAddress = Form.useWatch(['settings', 'address'], form) as string | undefined;
  const obfuscationJc = Form.useWatch(['settings', 'jc'], form) as number | undefined;
  const obfuscationDns = Form.useWatch(['settings', 'dns'], form) as string | undefined;
  const obfuscationH1 = Form.useWatch(['settings', 'h1'], form) as string | undefined;
  const obfuscationI1 = Form.useWatch(['settings', 'i1'], form) as string | undefined;

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
      if (msg?.success && msg.obj) {
        applyProvision(msg.obj as AwgProvisionResult, keepRemark);
        return;
      }
      applyProvision(buildLocalAwgProvision(), keepRemark);
      messageApi.warning('Panel host could not reach awg runtime — generated AWG plan locally');
    } catch {
      applyProvision(buildLocalAwgProvision(), keepRemark);
      messageApi.warning('Panel host could not reach awg runtime — generated AWG plan locally');
    } finally {
      setLoading(false);
    }
  }, [applyProvision, messageApi]);

  useEffect(() => {
    if (!isCreate) return;
    void loadProvision(true);
    void HttpUtil.get('/panel/api/awg/discovered', undefined, { silent: true })
      .then((msg) => {
        setDiscovered(Array.isArray(msg?.obj) ? msg.obj as AwgDiscovered[] : []);
      });
  }, [isCreate, loadProvision]);

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
      {isCreate && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="AWGv2 auto-provision"
          description={(
            <>
              The panel assigns the next free kernel interface (awg0, awg1, …), a random UDP port,
              subnet, keys, and full AWG 2.0 obfuscation (H1–H4 ranges, S1–S4, Jc/Jmin/Jmax, CPS I1–I5).
              Only the inbound remark is required before save.
            </>
          )}
        />
      )}

      {isCreate && provision && (
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
          <Descriptions.Item label="DNS">{obfuscationDns || provision.settings.dns as string}</Descriptions.Item>
          <Descriptions.Item label="Obfuscation">
            Jc={String(obfuscationJc ?? provision.settings.jc)}
            {' · '}
            H1={obfuscationH1 || String(provision.settings.h1)}
            {' · '}
            CPS I1={obfuscationI1 || String(provision.settings.i1)}
          </Descriptions.Item>
          <Descriptions.Item label="Config file">{provision.configPath}</Descriptions.Item>
          <Descriptions.Item label="Public key">{provision.publicKey}</Descriptions.Item>
        </Descriptions>
      )}

      {isCreate && (
        <Space style={{ marginBottom: 16 }}>
          <Button loading={loading} icon={<ReloadOutlined />} onClick={() => void loadProvision(true)}>
            Regenerate plan
          </Button>
        </Space>
      )}

      <Collapse
        defaultActiveKey={isCreate ? ['runtime', 'obfuscation'] : ['runtime']}
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
                  extra="Sequential names awg0, awg1, awg2 — assigned automatically on create"
                >
                  <Input placeholder="awg0" disabled={isCreate} />
                </Form.Item>
                <Form.Item name="port" label="Listen port (UDP)" hidden={isCreate}>
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                </Form.Item>
                <Form.Item name={['settings', 'mtu']} label="MTU">
                  <InputNumber min={1280} style={{ width: '100%' }} />
                </Form.Item>
                <Form.Item name={['settings', 'dns']} label={t('pages.inbounds.info.dns')}>
                  <Input placeholder="1.1.1.1,2606:4700:4700::1111" />
                </Form.Item>
                <Form.Item name={['settings', 'externalInterface']} label="External interface">
                  <Input placeholder="eth0" />
                </Form.Item>
              </>
            ),
          },
          {
            key: 'obfuscation',
            label: 'AWGv2 obfuscation (auto-generated)',
            children: (
              <>
                <Alert
                  type="success"
                  showIcon
                  style={{ marginBottom: 12 }}
                  message="Random AWG 2.0 profile"
                  description="H1–H4 are non-overlapping header ranges, S1–S4 unique padding, Jc junk train, I1–I5 CPS chain. Regenerate plan to roll new values."
                />
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
          ...(isCreate && pendingAdopt.length > 0 ? [{
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

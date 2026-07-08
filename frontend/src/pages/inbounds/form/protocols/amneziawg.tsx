import { Col, Form, Input, InputNumber, Row } from 'antd';

import WireguardFields from './wireguard';

interface AmneziawgFieldsProps {
  wgPubKey: string;
  regenInboundWg: () => void;
}

export default function AmneziawgFields(props: AmneziawgFieldsProps) {
  return (
    <>
      <WireguardFields {...props} />

      <Form.Item name={['settings', 'address']} label="Address">
        <Input placeholder="10.66.66.1/24" />
      </Form.Item>

      <Form.Item name={['settings', 'externalInterface']} label="External interface">
        <Input placeholder="eth0" />
      </Form.Item>

      <Row gutter={12}>
        <Col xs={24} md={8}>
          <Form.Item name={['settings', 'jc']} label="Jc">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item name={['settings', 'jmin']} label="Jmin">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={8}>
          <Form.Item name={['settings', 'jmax']} label="Jmax">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={12}>
        <Col xs={24} md={12}>
          <Form.Item name={['settings', 's1']} label="S1">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name={['settings', 's2']} label="S2">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={12}>
        <Col xs={24} md={12}>
          <Form.Item name={['settings', 's3']} label="S3">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
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

      <Form.Item name={['settings', 'postUp']} label="PostUp">
        <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
      </Form.Item>

      <Form.Item name={['settings', 'postDown']} label="PostDown">
        <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
      </Form.Item>
    </>
  );
}

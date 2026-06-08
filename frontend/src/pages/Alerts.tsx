import { useEffect, useState } from 'react';
import { Table, Select, Button, Tag, message, Space } from 'antd';
import { CheckCircleOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { getAlerts, resolveAlert } from '../api/alerts';
import { Alert } from '../types';

const SEVERITY_MAP: Record<string, { label: string; color: string; tagColor: string }> = {
  critical: { label: '严重', color: '#ff4d4f', tagColor: 'red' },
  warning: { label: '警告', color: '#faad14', tagColor: 'orange' },
  info: { label: '信息', color: '#1890ff', tagColor: 'blue' },
};

const TYPE_MAP: Record<string, string> = {
  time_sync: '时间同步',
  annotation_quality: '标注质量',
};

const Alerts: React.FC = () => {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(false);
  const [typeFilter, setTypeFilter] = useState<string | undefined>(undefined);
  const [severityFilter, setSeverityFilter] = useState<string | undefined>(undefined);

  const fetchAlerts = async () => {
    setLoading(true);
    try {
      const res = await getAlerts({
        type: typeFilter,
        severity: severityFilter,
      });
      setAlerts(res.data);
    } catch {
      message.error('获取告警列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAlerts();
  }, [typeFilter, severityFilter]);

  const handleResolve = async (id: number) => {
    try {
      await resolveAlert(id);
      message.success('告警已解决');
      fetchAlerts();
    } catch {
      message.error('操作失败');
    }
  };

  const columns: ColumnsType<Alert> = [
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => TYPE_MAP[type] || type,
    },
    {
      title: '场景',
      dataIndex: 'scene_name',
      key: 'scene_name',
      ellipsis: true,
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => {
        const s = SEVERITY_MAP[severity] || { label: severity, tagColor: 'default' };
        return <Tag color={s.tagColor}>{s.label}</Tag>;
      },
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (v: string) => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '状态',
      dataIndex: 'resolved',
      key: 'resolved',
      width: 100,
      render: (resolved: boolean) =>
        resolved ? (
          <Tag color="green" icon={<CheckCircleOutlined />}>已解决</Tag>
        ) : (
          <Tag color="volcano">未解决</Tag>
        ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) =>
        !record.resolved ? (
          <Button
            size="small"
            type="primary"
            onClick={() => handleResolve(record.id)}
          >
            解决
          </Button>
        ) : (
          <span style={{ color: '#999' }}>-</span>
        ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', gap: 16, alignItems: 'center' }}>
        <span>筛选:</span>
        <Select
          placeholder="告警类型"
          value={typeFilter}
          onChange={setTypeFilter}
          style={{ width: 150 }}
          allowClear
          options={[
            { label: '时间同步', value: 'time_sync' },
            { label: '标注质量', value: 'annotation_quality' },
          ]}
        />
        <Select
          placeholder="严重程度"
          value={severityFilter}
          onChange={setSeverityFilter}
          style={{ width: 150 }}
          allowClear
          options={[
            { label: '严重', value: 'critical' },
            { label: '警告', value: 'warning' },
            { label: '信息', value: 'info' },
          ]}
        />
      </div>

      <Table
        columns={columns}
        dataSource={alerts}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 10 }}
        rowClassName={(record) => {
          if (record.resolved) return '';
          return record.severity === 'critical'
            ? 'alert-row-critical'
            : record.severity === 'warning'
              ? 'alert-row-warning'
              : '';
        }}
      />

      <style>{`
        .alert-row-critical {
          background-color: #fff1f0 !important;
        }
        .alert-row-critical:hover > td {
          background-color: #ffccc7 !important;
        }
        .alert-row-warning {
          background-color: #fff7e6 !important;
        }
        .alert-row-warning:hover > td {
          background-color: #ffe58f !important;
        }
      `}</style>
    </div>
  );
};

export default Alerts;

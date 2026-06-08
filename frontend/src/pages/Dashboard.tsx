import { useEffect, useState } from 'react';
import { Card, Col, Row, List, Tag, Spin, message } from 'antd';
import {
  VideoCameraOutlined,
  CheckCircleOutlined,
  PercentageOutlined,
  AlertOutlined,
} from '@ant-design/icons';
import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { getSceneStats } from '../api/scenes';
import { getAlerts } from '../api/alerts';
import { SceneStats, Alert } from '../types';

const TYPE_COLORS: Record<string, string> = {
  highway: '#1890ff',
  urban: '#52c41a',
  rural: '#faad14',
  parking: '#722ed1',
};

const TYPE_LABELS: Record<string, string> = {
  highway: '高速',
  urban: '城市',
  rural: '乡村',
  parking: '停车场',
};

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<SceneStats | null>(null);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statsRes, alertsRes] = await Promise.all([
          getSceneStats(),
          getAlerts(),
        ]);
        setStats(statsRes.data);
        setAlerts(alertsRes.data.slice(0, 5));
      } catch {
        message.error('获取统计数据失败');
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
      </div>
    );
  }

  const pieData = (stats?.type_distribution || []).map((item) => ({
    name: TYPE_LABELS[item.type] || item.type,
    value: item.count,
    color: TYPE_COLORS[item.type] || '#999',
  }));

  const barData = stats?.scenes_per_month || [];

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <VideoCameraOutlined style={{ fontSize: 36, color: '#1890ff' }} />
              <div>
                <div style={{ fontSize: 14, color: '#999' }}>总场景数</div>
                <div style={{ fontSize: 28, fontWeight: 'bold' }}>
                  {stats?.total_scenes ?? 0}
                </div>
              </div>
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <CheckCircleOutlined style={{ fontSize: 36, color: '#52c41a' }} />
              <div>
                <div style={{ fontSize: 14, color: '#999' }}>已标注场景</div>
                <div style={{ fontSize: 28, fontWeight: 'bold' }}>
                  {stats?.annotated_scenes ?? 0}
                </div>
              </div>
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <PercentageOutlined style={{ fontSize: 36, color: '#faad14' }} />
              <div>
                <div style={{ fontSize: 14, color: '#999' }}>标注完成率</div>
                <div style={{ fontSize: 28, fontWeight: 'bold' }}>
                  {((stats?.annotation_rate ?? 0) * 100).toFixed(1)}%
                </div>
              </div>
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <AlertOutlined style={{ fontSize: 36, color: '#ff4d4f' }} />
              <div>
                <div style={{ fontSize: 14, color: '#999' }}>告警数量</div>
                <div style={{ fontSize: 28, fontWeight: 'bold' }}>
                  {stats?.alert_count ?? 0}
                </div>
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={12}>
          <Card title="场景类型分布">
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={pieData}
                  cx="50%"
                  cy="50%"
                  outerRadius={100}
                  dataKey="value"
                  label={({ name, value }) => `${name}: ${value}`}
                >
                  {pieData.map((entry, index) => (
                    <Cell key={index} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col span={12}>
          <Card title="每月新增场景">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={barData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="month" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="count" name="场景数" fill="#1890ff" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      <Card title="最近告警">
        <List
          dataSource={alerts}
          renderItem={(alert) => (
            <List.Item
              actions={[
                <Tag
                  key="severity"
                  color={
                    alert.severity === 'critical'
                      ? 'red'
                      : alert.severity === 'warning'
                        ? 'orange'
                        : 'blue'
                  }
                >
                  {alert.severity === 'critical'
                    ? '严重'
                    : alert.severity === 'warning'
                      ? '警告'
                      : '信息'}
                </Tag>,
              ]}
            >
              <List.Item.Meta
                title={alert.message}
                description={`${alert.scene_name} - ${new Date(alert.created_at).toLocaleString('zh-CN')}`}
              />
            </List.Item>
          )}
        />
      </Card>
    </div>
  );
};

export default Dashboard;

import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Table,
  Button,
  Input,
  Select,
  Modal,
  Form,
  Space,
  Dropdown,
  message,
  Tag,
  Upload,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ExportOutlined,
  CloudUploadOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { getScenes, uploadScene, deleteScene } from '../api/scenes';
import { exportScene } from '../api/export';
import { Scene } from '../types';

const TYPE_MAP: Record<string, string> = {
  highway: '高速',
  urban: '城市',
  rural: '乡村',
  parking: '停车场',
};

const STATUS_MAP: Record<string, { label: string; color: string }> = {
  uploaded: { label: '已上传', color: 'blue' },
  processing: { label: '处理中', color: 'orange' },
  ready: { label: '就绪', color: 'green' },
  error: { label: '错误', color: 'red' },
};

const SceneList: React.FC = () => {
  const navigate = useNavigate();
  const [scenes, setScenes] = useState<Scene[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<string | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const fetchScenes = async () => {
    setLoading(true);
    try {
      const res = await getScenes({ type: typeFilter, search });
      setScenes(res.data);
    } catch {
      message.error('获取场景列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchScenes();
  }, [search, typeFilter]);

  const handleUpload = async (values: {
    name: string;
    description: string;
    type: string;
  }) => {
    try {
      const formData = new FormData();
      formData.append('name', values.name);
      formData.append('description', values.description);
      formData.append('type', values.type);
      formData.append('file', new Blob([]), 'scene.bag');
      await uploadScene(formData);
      message.success('场景上传成功');
      setModalOpen(false);
      form.resetFields();
      fetchScenes();
    } catch {
      message.error('上传失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteScene(id);
      message.success('删除成功');
      fetchScenes();
    } catch {
      message.error('删除失败');
    }
  };

  const handleExport = async (sceneId: number, format: 'openscenario' | 'rosbag') => {
    try {
      await exportScene(sceneId, format);
      message.success(`导出${format === 'openscenario' ? 'OpenSCENARIO' : 'ROS bag'}任务已创建`);
    } catch {
      message.error('导出失败');
    }
  };

  const columns: ColumnsType<Scene> = [
    {
      title: '场景名称',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => TYPE_MAP[type] || type,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const s = STATUS_MAP[status] || { label: status, color: 'default' };
        return <Tag color={s.color}>{s.label}</Tag>;
      },
    },
    {
      title: '时长(秒)',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (v: number) => v.toFixed(1),
    },
    {
      title: '帧数',
      dataIndex: 'frame_count',
      key: 'frame_count',
      width: 80,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (v: string) => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space>
          <Button
            size="small"
            type="link"
            onClick={() => navigate(`/scenes/${record.id}/replay`)}
          >
            回放
          </Button>
          <Button
            size="small"
            type="link"
            onClick={() => navigate(`/scenes/${record.id}/annotate`)}
          >
            标注
          </Button>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'openscenario',
                  label: 'OpenSCENARIO',
                  onClick: () => handleExport(record.id, 'openscenario'),
                },
                {
                  key: 'rosbag',
                  label: 'ROS bag',
                  onClick: () => handleExport(record.id, 'rosbag'),
                },
              ],
            }}
          >
            <Button size="small" icon={<ExportOutlined />}>
              导出
            </Button>
          </Dropdown>
          <Button
            size="small"
            danger
            type="link"
            onClick={() => handleDelete(record.id)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', gap: 16, alignItems: 'center' }}>
        <Input
          placeholder="搜索场景名称"
          prefix={<SearchOutlined />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ width: 300 }}
          allowClear
        />
        <Select
          placeholder="筛选类型"
          value={typeFilter}
          onChange={setTypeFilter}
          style={{ width: 150 }}
          allowClear
          options={[
            { label: '高速', value: 'highway' },
            { label: '城市', value: 'urban' },
            { label: '乡村', value: 'rural' },
            { label: '停车场', value: 'parking' },
          ]}
        />
        <div style={{ flex: 1 }} />
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setModalOpen(true)}
        >
          上传场景
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={scenes}
        rowKey="id"
        loading={loading}
        onRow={(record) => ({
          onDoubleClick: () => navigate(`/scenes/${record.id}/replay`),
          style: { cursor: 'pointer' },
        })}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title="上传场景"
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          form.resetFields();
        }}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleUpload}>
          <Form.Item
            name="name"
            label="场景名称"
            rules={[{ required: true, message: '请输入场景名称' }]}
          >
            <Input placeholder="请输入场景名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入场景描述" />
          </Form.Item>
          <Form.Item
            name="type"
            label="类型"
            rules={[{ required: true, message: '请选择场景类型' }]}
          >
            <Select
              placeholder="选择场景类型"
              options={[
                { label: '高速', value: 'highway' },
                { label: '城市', value: 'urban' },
                { label: '乡村', value: 'rural' },
                { label: '停车场', value: 'parking' },
              ]}
            />
          </Form.Item>
          <Form.Item name="file" label="场景文件">
            <Upload beforeUpload={() => false} maxCount={1}>
              <Button icon={<CloudUploadOutlined />}>选择文件</Button>
            </Upload>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default SceneList;

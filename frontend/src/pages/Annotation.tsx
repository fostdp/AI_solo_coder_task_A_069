import { useEffect, useRef, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Card,
  Button,
  Space,
  Select,
  InputNumber,
  List,
  Tag,
  message,
  Row,
  Col,
  Spin,
} from 'antd';
import {
  ArrowLeftOutlined,
  StepBackwardOutlined,
  StepForwardOutlined,
  DeleteOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import { getScene } from '../api/scenes';
import { getAnnotationsByScene, createAnnotation, deleteAnnotation } from '../api/annotations';
import { Annotation, BoundingBox } from '../types';

const LABEL_OPTIONS = [
  { label: '车辆', value: 'vehicle' },
  { label: '行人', value: 'pedestrian' },
  { label: '自行车', value: 'bicycle' },
  { label: '其他', value: 'other' },
];

const LABEL_COLORS: Record<string, string> = {
  vehicle: '#1890ff',
  pedestrian: '#52c41a',
  bicycle: '#faad14',
  other: '#722ed1',
};

const LABEL_MAP: Record<string, string> = {
  vehicle: '车辆',
  pedestrian: '行人',
  bicycle: '自行车',
  other: '其他',
};

const AnnotationPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const sceneId = Number(id);

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [sceneName, setSceneName] = useState('');
  const [frameCount, setFrameCount] = useState(0);
  const [currentFrame, setCurrentFrame] = useState(0);
  const [annotations, setAnnotations] = useState<Annotation[]>([]);
  const [currentLabel, setCurrentLabel] = useState<string>('vehicle');
  const [trackId, setTrackId] = useState<number>(1);
  const [drawing, setDrawing] = useState(false);
  const [drawStart, setDrawStart] = useState<{ x: number; y: number } | null>(null);
  const [currentBox, setCurrentBox] = useState<BoundingBox | null>(null);
  const [boxes, setBoxes] = useState<BoundingBox[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchAnnotations = useCallback(async () => {
    try {
      const res = await getAnnotationsByScene(sceneId);
      setAnnotations(res.data);
    } catch {
      message.error('获取标注数据失败');
    }
  }, [sceneId]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const sceneRes = await getScene(sceneId);
        setSceneName(sceneRes.data.name);
        setFrameCount(sceneRes.data.frame_count);
      } catch {
        message.error('获取场景数据失败');
      } finally {
        setLoading(false);
      }
    };
    fetchData();
    fetchAnnotations();
  }, [sceneId, fetchAnnotations]);

  const frameAnnotations = annotations.filter((a) => a.frame_index === currentFrame);

  useEffect(() => {
    const allBoxes = frameAnnotations.flatMap((a) => a.bounding_boxes);
    setBoxes(allBoxes);
  }, [currentFrame, annotations]);

  const drawCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    ctx.fillStyle = '#2a2a3e';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    ctx.fillStyle = '#3a3a5e';
    ctx.font = '14px monospace';
    ctx.fillText(`Frame ${currentFrame + 1}`, 10, 20);

    const allBoxes = [...boxes];
    if (currentBox) {
      allBoxes.push(currentBox);
    }

    allBoxes.forEach((box) => {
      const color = LABEL_COLORS[box.label] || '#fff';
      ctx.strokeStyle = color;
      ctx.lineWidth = 2;
      ctx.strokeRect(box.x, box.y, box.width, box.height);
      ctx.fillStyle = color;
      ctx.font = '12px sans-serif';
      ctx.fillText(
        `${LABEL_MAP[box.label] || box.label} #${box.track_id}`,
        box.x,
        box.y - 5,
      );
    });
  }, [boxes, currentBox, currentFrame]);

  useEffect(() => {
    drawCanvas();
  }, [drawCanvas]);

  const getCanvasCoords = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) return { x: 0, y: 0 };
    const rect = canvas.getBoundingClientRect();
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;
    return {
      x: (e.clientX - rect.left) * scaleX,
      y: (e.clientY - rect.top) * scaleY,
    };
  };

  const handleMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const coords = getCanvasCoords(e);
    setDrawing(true);
    setDrawStart(coords);
  };

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!drawing || !drawStart) return;
    const coords = getCanvasCoords(e);
    setCurrentBox({
      x: Math.min(drawStart.x, coords.x),
      y: Math.min(drawStart.y, coords.y),
      width: Math.abs(coords.x - drawStart.x),
      height: Math.abs(coords.y - drawStart.y),
      label: currentLabel,
      track_id: trackId,
      frame_index: currentFrame,
    });
  };

  const handleMouseUp = () => {
    if (currentBox && currentBox.width > 5 && currentBox.height > 5) {
      setBoxes((prev) => [...prev, currentBox]);
    }
    setDrawing(false);
    setDrawStart(null);
    setCurrentBox(null);
  };

  const handleSave = async () => {
    try {
      await createAnnotation({
        scene_id: sceneId,
        frame_index: currentFrame,
        bounding_boxes: boxes,
      });
      message.success('标注保存成功');
      fetchAnnotations();
    } catch {
      message.error('保存失败');
    }
  };

  const handleDeleteAnnotation = async (annotationId: number) => {
    try {
      await deleteAnnotation(annotationId);
      message.success('标注已删除');
      fetchAnnotations();
    } catch {
      message.error('删除失败');
    }
  };

  const handleDeleteBox = (index: number) => {
    setBoxes((prev) => prev.filter((_, i) => i !== index));
  };

  const goToFrame = (frame: number) => {
    if (frame >= 0 && frame < frameCount) {
      setCurrentFrame(frame);
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/scenes')}>
          返回列表
        </Button>
        <span style={{ marginLeft: 16, fontSize: 18, fontWeight: 'bold' }}>
          {sceneName} - 标注
        </span>
      </div>

      <Row gutter={16}>
        <Col span={16}>
          <Card>
            <canvas
              ref={canvasRef}
              width={800}
              height={500}
              style={{
                width: '100%',
                border: '1px solid #444',
                borderRadius: 8,
                cursor: 'crosshair',
              }}
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
            />
            <div style={{ marginTop: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Space>
                <Button
                  icon={<StepBackwardOutlined />}
                  onClick={() => goToFrame(currentFrame - 1)}
                  disabled={currentFrame <= 0}
                >
                  上一帧
                </Button>
                <span>第 {currentFrame + 1} 帧 / 共 {frameCount} 帧</span>
                <Button
                  icon={<StepForwardOutlined />}
                  onClick={() => goToFrame(currentFrame + 1)}
                  disabled={currentFrame >= frameCount - 1}
                >
                  下一帧
                </Button>
                <InputNumber
                  min={1}
                  max={frameCount}
                  value={currentFrame + 1}
                  onChange={(val) => val && goToFrame(val - 1)}
                />
              </Space>
              <Button type="primary" icon={<SaveOutlined />} onClick={handleSave}>
                保存标注
              </Button>
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="标注工具" style={{ marginBottom: 16 }}>
            <div style={{ marginBottom: 12 }}>
              <span style={{ marginRight: 8 }}>标签:</span>
              <Select
                value={currentLabel}
                onChange={setCurrentLabel}
                style={{ width: 150 }}
                options={LABEL_OPTIONS}
              />
            </div>
            <div>
              <span style={{ marginRight: 8 }}>Track ID:</span>
              <InputNumber
                min={1}
                value={trackId}
                onChange={(val) => setTrackId(val || 1)}
              />
            </div>
          </Card>

          <Card title={`当前帧标注 (${boxes.length})`}>
            <List
              dataSource={boxes}
              renderItem={(box, index) => (
                <List.Item
                  actions={[
                    <Button
                      key="del"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => handleDeleteBox(index)}
                    />,
                  ]}
                >
                  <Space>
                    <Tag color={LABEL_COLORS[box.label]}>
                      {LABEL_MAP[box.label] || box.label}
                    </Tag>
                    <span>Track #{box.track_id}</span>
                  </Space>
                </List.Item>
              )}
              locale={{ emptyText: '暂无标注' }}
            />
          </Card>

          <Card title="历史标注" style={{ marginTop: 16 }}>
            <List
              dataSource={frameAnnotations}
              renderItem={(ann) => (
                <List.Item
                  actions={[
                    <Button
                      key="del"
                      size="small"
                      danger
                      onClick={() => handleDeleteAnnotation(ann.id)}
                    >
                      删除
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    title={`帧 ${ann.frame_index + 1} - ${ann.bounding_boxes.length} 个框`}
                    description={new Date(ann.updated_at).toLocaleString('zh-CN')}
                  />
                </List.Item>
              )}
              locale={{ emptyText: '暂无标注' }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default AnnotationPage;

import { useEffect, useRef, useState, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Slider, Button, Space, Row, Col, Descriptions, Spin, message } from 'antd';
import { PlayCircleOutlined, PauseCircleOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import { Canvas, useFrame } from '@react-three/fiber';
import { OrbitControls, Plane, Text } from '@react-three/drei';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { getReplayData, getCANSignals } from '../api/replay';
import { getScene } from '../api/scenes';
import { CANSignal } from '../types';

const TYPE_COLORS: Record<string, string> = {
  speed: '#1890ff',
  steering_angle: '#52c41a',
  throttle: '#faad14',
};

const TYPE_LABELS: Record<string, string> = {
  speed: '速度(km/h)',
  steering_angle: '转向角(°)',
  throttle: '油门(%)',
};

function ScenePlane({ frameIndex }: { frameIndex: number }) {
  const meshRef = useRef<any>(null);
  const hue = useMemo(() => (frameIndex * 3) % 360, [frameIndex]);

  useFrame(() => {
    if (meshRef.current) {
      meshRef.current.rotation.x = -Math.PI / 2;
    }
  });

  return (
    <group>
      <Plane ref={meshRef} args={[10, 6]} position={[0, 0, 0]}>
        <meshStandardMaterial
          color={`hsl(${hue}, 60%, 50%)`}
          toneMapped={false}
        />
      </Plane>
      {[
        { pos: [-3, 0.5, -2] as [number, number, number], color: '#4CAF50', label: 'Ego' },
        { pos: [1, 0.5, -1] as [number, number, number], color: '#FF9800', label: 'V1' },
        { pos: [-1, 0.5, 1] as [number, number, number], color: '#2196F3', label: 'V2' },
        { pos: [3, 0.5, 0.5] as [number, number, number], color: '#9C27B0', label: 'P1' },
      ].map((obj, i) => (
        <group key={i} position={obj.pos}>
          <mesh>
            <boxGeometry args={[0.4, 0.3, 0.6]} />
            <meshStandardMaterial color={obj.color} />
          </mesh>
          <Text
            position={[0, 0.4, 0]}
            fontSize={0.2}
            color="white"
            anchorX="center"
            anchorY="middle"
          >
            {obj.label}
          </Text>
        </group>
      ))}
    </group>
  );
}

function SceneCanvas({ frameIndex }: { frameIndex: number }) {
  return (
    <Canvas camera={{ position: [0, 5, 8], fov: 50 }}>
      <ambientLight intensity={0.5} />
      <directionalLight position={[5, 10, 5]} intensity={1} />
      <ScenePlane frameIndex={frameIndex} />
      <OrbitControls />
      <gridHelper args={[20, 20, '#444', '#222']} />
    </Canvas>
  );
}

const SceneReplay: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const sceneId = Number(id);

  const [sceneName, setSceneName] = useState('');
  const [frameCount, setFrameCount] = useState(0);
  const [duration, setDuration] = useState(0);
  const [fps, setFps] = useState(30);
  const [currentFrame, setCurrentFrame] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [canSignals, setCanSignals] = useState<CANSignal[]>([]);
  const [loading, setLoading] = useState(true);
  const animRef = useRef<number | null>(null);
  const lastTimeRef = useRef<number>(0);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [sceneRes, replayRes, canRes] = await Promise.all([
          getScene(sceneId),
          getReplayData(sceneId),
          getCANSignals(sceneId),
        ]);
        setSceneName(sceneRes.data.name);
        setFrameCount(replayRes.data.frame_count);
        setDuration(replayRes.data.duration);
        setFps(replayRes.data.fps);
        setCanSignals(canRes.data);
      } catch {
        message.error('获取回放数据失败');
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [sceneId]);

  useEffect(() => {
    if (playing && frameCount > 0) {
      const interval = 1000 / fps;
      const animate = (time: number) => {
        if (time - lastTimeRef.current >= interval) {
          lastTimeRef.current = time;
          setCurrentFrame((prev) => {
            if (prev >= frameCount - 1) {
              setPlaying(false);
              return prev;
            }
            return prev + 1;
          });
        }
        animRef.current = requestAnimationFrame(animate);
      };
      animRef.current = requestAnimationFrame(animate);
    }
    return () => {
      if (animRef.current !== null) {
        cancelAnimationFrame(animRef.current);
      }
    };
  }, [playing, frameCount, fps]);

  const currentTime = fps > 0 ? (currentFrame / fps).toFixed(2) : '0.00';

  const currentSignal = canSignals[currentFrame] || canSignals[canSignals.length - 1] || {
    speed: 0,
    steering_angle: 0,
    throttle: 0,
    brake: 0,
  };

  const chartData = canSignals.map((sig, idx) => ({
    time: Number((idx / fps).toFixed(2)),
    speed: sig.speed,
    steering_angle: sig.steering_angle,
    throttle: sig.throttle,
  }));

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
          {sceneName} - 场景回放
        </span>
      </div>

      <Row gutter={16}>
        <Col span={16}>
          <Card>
            <div style={{ height: 400, background: '#1a1a2e', borderRadius: 8 }}>
              <SceneCanvas frameIndex={currentFrame} />
            </div>
            <div style={{ marginTop: 16 }}>
              <Slider
                min={0}
                max={Math.max(frameCount - 1, 0)}
                value={currentFrame}
                onChange={(val) => {
                  setCurrentFrame(val);
                  setPlaying(false);
                }}
              />
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Space>
                  <Button
                    type="primary"
                    shape="circle"
                    icon={playing ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                    onClick={() => setPlaying(!playing)}
                  />
                  <span>当前时间: {currentTime}s</span>
                  <span>帧: {currentFrame + 1} / {frameCount}</span>
                </Space>
                <span>总时长: {duration.toFixed(1)}s</span>
              </div>
            </div>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="CAN信号" style={{ marginBottom: 16 }}>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="速度">{currentSignal.speed.toFixed(1)} km/h</Descriptions.Item>
              <Descriptions.Item label="转向角">{currentSignal.steering_angle.toFixed(1)}°</Descriptions.Item>
              <Descriptions.Item label="油门">{currentSignal.throttle.toFixed(1)}%</Descriptions.Item>
              <Descriptions.Item label="制动">{currentSignal.brake.toFixed(1)}%</Descriptions.Item>
            </Descriptions>
          </Card>
          <Card title="CAN信号曲线">
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" label={{ value: '时间(s)', position: 'insideBottom', offset: -5 }} />
                <YAxis />
                <Tooltip />
                <Legend />
                <Line type="monotone" dataKey="speed" stroke={TYPE_COLORS.speed} name={TYPE_LABELS.speed} dot={false} />
                <Line type="monotone" dataKey="steering_angle" stroke={TYPE_COLORS.steering_angle} name={TYPE_LABELS.steering_angle} dot={false} />
                <Line type="monotone" dataKey="throttle" stroke={TYPE_COLORS.throttle} name={TYPE_LABELS.throttle} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default SceneReplay;

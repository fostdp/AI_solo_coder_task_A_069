import { useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu } from 'antd';
import {
  DashboardOutlined,
  VideoCameraOutlined,
  AlertOutlined,
} from '@ant-design/icons';

const { Sider } = Layout;

const Sidebar: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  const selectedKey = (() => {
    if (location.pathname === '/') return '/';
    if (location.pathname.startsWith('/scenes')) return '/scenes';
    if (location.pathname.startsWith('/alerts')) return '/alerts';
    return '/';
  })();

  return (
    <Sider collapsible theme="dark">
      <div
        style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#fff',
          fontSize: 16,
          fontWeight: 'bold',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
        }}
      >
        ADAS场景库
      </div>
      <Menu
        theme="dark"
        mode="inline"
        selectedKeys={[selectedKey]}
        onClick={({ key }) => navigate(key)}
        items={[
          {
            key: '/',
            icon: <DashboardOutlined />,
            label: '仪表盘',
          },
          {
            key: '/scenes',
            icon: <VideoCameraOutlined />,
            label: '场景管理',
          },
          {
            key: '/alerts',
            icon: <AlertOutlined />,
            label: '告警中心',
          },
        ]}
      />
    </Sider>
  );
};

export default Sidebar;

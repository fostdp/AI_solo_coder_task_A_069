import { Routes, Route } from 'react-router-dom';
import { Layout } from 'antd';
import Sidebar from './components/Sidebar';
import Dashboard from './pages/Dashboard';
import SceneList from './pages/SceneList';
import SceneReplay from './pages/SceneReplay';
import Annotation from './pages/Annotation';
import Alerts from './pages/Alerts';

const { Content } = Layout;

const App: React.FC = () => {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sidebar />
      <Layout>
        <Content style={{ margin: '16px' }}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/scenes" element={<SceneList />} />
            <Route path="/scenes/:id/replay" element={<SceneReplay />} />
            <Route path="/scenes/:id/annotate" element={<Annotation />} />
            <Route path="/alerts" element={<Alerts />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  );
};

export default App;

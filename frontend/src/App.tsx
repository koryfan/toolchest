import {useEffect, type ComponentType} from 'react';
import {Tools} from '../bindings/toolchest/internal/services/appservice';
import Sidebar from './components/Sidebar';
import Toolbar from './components/Toolbar';
import {NoticeProvider} from './components/Notice';
import JsonPage from './pages/JsonPage';
import TimePage from './pages/TimePage';
import TunnelPage from './pages/TunnelPage';
import DbeaverPage from './pages/DbeaverPage';
import UuidPage from './pages/UuidPage';
import {useStore, initTunnelEvents, initDbeaverRestore} from './store';
import styles from './App.module.css';

// 内容区注册表：Go 下发的 Meta.ContentType → 前端组件。
// 纯 UI 形态在前端定（CodeMirror/树/输入行无法配置化），
// 配置/操作/内容全部来自 Go。新增工具：Go 侧加包 + 注册表加一行 +
// （仅当有新内容形态时）这里加一行映射。
const contentRegistry: Record<string, ComponentType> = {
  json: JsonPage,
  time: TimePage,
  tunnel: TunnelPage,
  dbeaver: DbeaverPage,
  uuid: UuidPage,
};

// 图标名 → SVG URL（Meta.Icon 下发的是图标名）
const iconUrls: Record<string, [white: string, gray: string]> = {
  json: ['/icons/code-white.svg', '/icons/code-gray.svg'],
  time: ['/icons/clock-white.svg', '/icons/clock-gray.svg'],
  tunnel: ['/icons/tunnel-white.svg', '/icons/tunnel-gray.svg'],
  dbeaver: ['/icons/db-white.svg', '/icons/db-gray.svg'],
  uuid: ['/icons/hash-white.svg', '/icons/hash-gray.svg'],
};

function App() {
  const descs = useStore(s => s.descs);
  const activeId = useStore(s => s.activeId);
  const setDescs = useStore(s => s.setDescs);
  const setActiveId = useStore(s => s.setActiveId);
  const flush = useStore(s => s.flush);

  const tunnelData = useStore(s => s.tunnelData);

  // 启动拉取工具配置（Go 唯一权威）+ 隧道事件订阅
  useEffect(() => {
    initTunnelEvents();
    initDbeaverRestore();
    void Tools().then(list => {
      if (list && list.length > 0) {
        setDescs(list);
        for (const d of list) {
          useStore.getState().setActions(d.meta.id, d.actions ?? []);
        }
        setActiveId(list[0].meta.id);
      }
    });
  }, [setDescs, setActiveId]);

  const active = descs.find(d => d.meta.id === activeId);
  const actions = useStore(s => s.actions[activeId]) ?? active?.actions ?? [];
  const Content = contentRegistry[active?.meta.contentType ?? ''];

  return (
    <NoticeProvider>
      <div className={styles.root}>
        <Sidebar
          icons={descs.map(d => ({
            key: d.meta.id,
            url: (iconUrls[d.meta.icon] ?? ['', ''])[d.meta.id === activeId ? 0 : 1],
            title: d.meta.title,
            selected: d.meta.id === activeId,
            running: d.meta.id === 'tunnel' && tunnelData.running > 0,
            onClick: () => {
              flush(activeId); // 切走前把 pending 镜像推给 Go
              setActiveId(d.meta.id);
            },
          }))}
        />
        <div className={styles.main}>
          {active && (
            <>
              <Toolbar
                title={active.meta.title}
                toolID={active.meta.id}
                actions={actions}
              />
              {Content && <Content/>}
            </>
          )}
        </div>
      </div>
    </NoticeProvider>
  );
}

export default App;

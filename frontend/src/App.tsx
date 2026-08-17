import {useState} from 'react';
import Sidebar from './components/Sidebar';
import Toolbar from './components/Toolbar';
import {NoticeProvider} from './components/Notice';
import JsonPage, {JsonActions} from './pages/JsonPage';
import TimePage, {TimeActions} from './pages/TimePage';
import type {Tool} from './types';
import styles from './App.module.css';

// 工具注册表（对应旧 Fyne 版 pages.All）：顺序 = 侧栏顺序，第一个默认激活。
// 新增工具：pages/ 加页面 + 这里加一行。
const tools: Tool[] = [
  {
    id: 'json',
    title: 'JSON 工具',
    icon: sel => sel ? '/icons/code-white.svg' : '/icons/code-gray.svg',
    toolbarActions: JsonActions,
    content: JsonPage,
  },
  {
    id: 'time',
    title: '时间戳转换',
    icon: sel => sel ? '/icons/clock-white.svg' : '/icons/clock-gray.svg',
    toolbarActions: TimeActions,
    content: TimePage,
  },
];

function App() {
  const [activeId, setActiveId] = useState(tools[0].id);
  const active = tools.find(t => t.id === activeId)!;
  const Actions = active.toolbarActions;
  const Content = active.content;

  return (
    <NoticeProvider>
      <div className={styles.root}>
        <Sidebar
          icons={tools.map(t => ({
            key: t.id,
            url: t.icon(t.id === activeId),
            title: t.title,
            selected: t.id === activeId,
            onClick: () => setActiveId(t.id),
          }))}
        />
        <div className={styles.main}>
          <Toolbar title={active.title} actions={<Actions/>}/>
          <Content/>
        </div>
      </div>
    </NoticeProvider>
  );
}

export default App;

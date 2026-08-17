import {useState} from 'react';
import JsonEditor from '../components/JsonEditor';
import JsonTree from '../components/JsonTree';
import {useToolContent, useStore} from '../store';
import styles from './JsonPage.module.css';

// JSON 工具页：视图切换（代码/树形）+ 编辑器/树。
// 内容读写全走全局 store（Go 是权威，输入经 debounce 镜像同步；
// 工具栏操作结果经 replaceContent 回写 store，这里订阅即可）。
type ViewMode = 'code' | 'tree';

export default function JsonPage() {
  const [view, setView] = useState<ViewMode>('code');
  const input = useToolContent('json');
  const setContent = useStore(s => s.setContent);

  return (
    <div className={styles.page}>
      <div className={styles.viewTabs}>
        <button className={view === 'code' ? styles.tabActive : ''} onClick={() => setView('code')}>代码</button>
        <button className={view === 'tree' ? styles.tabActive : ''} onClick={() => setView('tree')}>树形</button>
      </div>
      {view === 'code' ? (
        <div className={styles.editorWrap}>
          <JsonEditor value={input} onChange={v => setContent('json', v)}/>
        </div>
      ) : (
        <JsonTree input={input}/>
      )}
    </div>
  );
}

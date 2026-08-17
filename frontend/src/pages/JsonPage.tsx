import {useRef, useState} from 'react';
import {JSONService} from '../../bindings/tool-dev/internal/services';
import {useNotice} from '../components/Notice';
import styles from './JsonPage.module.css';

// JSON 工具页：顶部「代码/树形」切换 + 输入区。
// 操作全走 Go 后端（internal/services），成功替换输入 + 绿提示，失败红提示。
// 工具栏操作区（JsonActions）在 Toolbar 里无法共享本组件 state，
// 通过 window 事件总线联动；inputRef 保证事件回调读到最新输入。
type ViewMode = 'code' | 'tree';
type Op = (s: string) => Promise<{ content: string; error: string }>;

export function JsonActions() {
  const [indent, setIndent] = useState('  ');
  const [menu, setMenu] = useState<'none' | 'setting' | 'more'>('none');

  return (
    <div className="toolbar-actions">
      <button onClick={() => window.dispatchEvent(new CustomEvent('json:clear'))}>清空</button>
      <button onClick={() => window.dispatchEvent(new CustomEvent('json:copy'))}>复制</button>
      <button onClick={() => window.dispatchEvent(new CustomEvent('json:format', {detail: indent}))}>格式化</button>

      <div className={styles.menuWrap}>
        <button onClick={() => setMenu(m => m === 'setting' ? 'none' : 'setting')}>设置 ▾</button>
        {menu === 'setting' && (
          <div className={styles.menu}>
            {([['2 空格', '  '], ['4 空格', '    '], ['Tab', '\t']] as const).map(([label, ind]) => (
              <div key={ind} onClick={() => { setIndent(ind); window.dispatchEvent(new CustomEvent('json:format', {detail: ind})); setMenu('none'); }}>{label}</div>
            ))}
          </div>
        )}
      </div>

      <div className={styles.menuWrap}>
        <button onClick={() => setMenu(m => m === 'more' ? 'none' : 'more')}>更多 ▾</button>
        {menu === 'more' && (
          <div className={styles.menu}>
            <div onClick={() => { window.dispatchEvent(new CustomEvent('json:op', {detail: 'minify'})); setMenu('none'); }}>压缩</div>
            <div onClick={() => { window.dispatchEvent(new CustomEvent('json:op', {detail: 'escape'})); setMenu('none'); }}>转义</div>
            <div onClick={() => { window.dispatchEvent(new CustomEvent('json:op', {detail: 'unescape'})); setMenu('none'); }}>反转义</div>
            <div onClick={() => { window.dispatchEvent(new CustomEvent('json:op', {detail: 'c2u'})); setMenu('none'); }}>中文 → Unicode</div>
            <div onClick={() => { window.dispatchEvent(new CustomEvent('json:op', {detail: 'u2c'})); setMenu('none'); }}>Unicode → 中文</div>
          </div>
        )}
      </div>
    </div>
  );
}

export default function JsonPage() {
  const [input, setInput] = useState('');
  const [view, setView] = useState<ViewMode>('code');
  const notice = useNotice();
  const inputRef = useRef('');
  inputRef.current = input; // 每次渲染同步，事件回调经 ref 读最新值

  const runOp = (fn: Op, okMsg: string) => {
    fn(inputRef.current).then(r => {
      if (r.error) {
        notice.showError(r.error);
      } else {
        setInput(r.content);
        notice.showSuccess(okMsg);
      }
    }).catch(e => notice.showError(String(e)));
  };

  // 工具栏操作事件总线（挂载一次，回调经 ref/handler 读最新状态）
  const handlersRef = useRef({clear: () => {}, copy: () => {}, op: (_name: string) => {}, format: (_indent: string) => {}});
  handlersRef.current = {
    clear: () => { setInput(''); notice.showSuccess('已清空'); },
    copy: () => {
      const text = inputRef.current;
      if (!text) { notice.showError('内容为空，无以复制'); return; }
      navigator.clipboard.writeText(text).then(
        () => notice.showSuccess(`已复制全部内容（${text.length} 字符）`),
        () => notice.showError('复制失败'));
    },
    format: (indent: string) => {
      if (!inputRef.current.trim()) return;
      JSONService.Pretty(inputRef.current, indent).then(r => {
        if (r.error) notice.showError(r.error);
        else { setInput(r.content); notice.showSuccess('格式化完成'); }
      });
    },
    op: (name: string) => {
      const ops: Record<string, [Op, string]> = {
        minify: [s => JSONService.Minify(s), '压缩完成'],
        escape: [s => JSONService.Escape(s), '转义完成'],
        unescape: [s => JSONService.Unescape(s), '反转义完成'],
        c2u: [s => JSONService.ChineseToUnicode(s), '中文→Unicode 完成'],
        u2c: [s => JSONService.UnicodeToChinese(s), 'Unicode→中文 完成'],
      };
      const op = ops[name];
      if (op) runOp(op[0], op[1]);
    },
  };
  if (!(window as any).__jsonHandlersBound) {
    (window as any).__jsonHandlersBound = true;
    window.addEventListener('json:clear', () => handlersRef.current.clear());
    window.addEventListener('json:copy', () => handlersRef.current.copy());
    window.addEventListener('json:op', (e: Event) => handlersRef.current.op((e as CustomEvent).detail));
    window.addEventListener('json:format', (e: Event) => handlersRef.current.format((e as CustomEvent).detail));
  }

  return (
    <div className={styles.page}>
      <div className={styles.viewTabs}>
        <button className={view === 'code' ? styles.tabActive : ''} onClick={() => setView('code')}>代码</button>
        <button className={view === 'tree' ? styles.tabActive : ''} onClick={() => setView('tree')}>树形</button>
      </div>
      {view === 'code' ? (
        <textarea
          className={`code ${styles.editor}`}
          value={input}
          onChange={e => setInput(e.target.value)}
          placeholder="粘贴或输入 JSON ..."
          spellCheck={false}
        />
      ) : (
        <div className={styles.treePlaceholder}>树视图（待实现）</div>
      )}
    </div>
  );
}

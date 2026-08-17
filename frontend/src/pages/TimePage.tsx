import {useRef, useState} from 'react';
import {TimeService} from '../../bindings/tool-dev/internal/services';
import type {ParseResult} from '../../bindings/tool-dev/internal/transform/models';
import {useNotice} from '../components/Notice';
import styles from './TimePage.module.css';

// 时间戳工具页：输入时间戳（秒/毫秒）或日期字符串 → 多格式结果。
// 解析走 Go 后端（internal/transform.TimeParser）。
// 工具栏「当前时间戳」填充当前秒级时间戳。

export function TimeActions() {
  return (
    <div className="toolbar-actions">
      <button onClick={() => window.dispatchEvent(new CustomEvent('time:now'))}>当前时间戳</button>
    </div>
  );
}

// 结果行：label + 值 + 点击复制
function ResultRow({label, value}: { label: string; value: string }) {
  const notice = useNotice();
  if (!value) return null;
  return (
    <div className={styles.row} onClick={() => {
      navigator.clipboard.writeText(value).then(
        () => notice.showSuccess(`已复制: ${value.length > 40 ? value.slice(0, 40) + '…' : value}`),
        () => notice.showError('复制失败'));
    }}>
      <span className={styles.label}>{label}</span>
      <span className={styles.value}>{value}</span>
    </div>
  );
}

export default function TimePage() {
  const [input, setInput] = useState('');
  const [result, setResult] = useState<ParseResult | null>(null);
  const notice = useNotice();

  const parse = (text: string) => {
    if (!text.trim()) { notice.showError('请输入时间戳或日期'); return; }
    TimeService.Parse(text.trim()).then(r => {
      if (r.Error) {
        setResult(null);
        notice.showError(r.Error);
      } else {
        setResult(r);
      }
    }).catch(e => notice.showError(String(e)));
  };

  // 工具栏「当前时间戳」按钮经事件总线联动（组件不在同一树，同 JsonPage 模式）
  const handlersRef = useRef({now: () => {}});
  handlersRef.current.now = () => {
    const ts = String(Math.floor(Date.now() / 1000));
    setInput(ts);
    parse(ts);
  };
  if (!(window as any).__timeHandlersBound) {
    (window as any).__timeHandlersBound = true;
    window.addEventListener('time:now', () => handlersRef.current.now());
  }

  return (
    <div className={styles.page}>
      <div className={styles.inputBar}>
        <input
          className={styles.input}
          value={input}
          placeholder="秒/毫秒时间戳或日期字符串，如 1720000000、1720000000000、2024-07-03 15:30:00"
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') parse(input); }}
          spellCheck={false}
        />
        <button className={styles.btn} onClick={() => parse(input)}>解析</button>
      </div>
      {result && (
        <div className={styles.results}>
          <ResultRow label="本地时间" value={result.LocalTime}/>
          <ResultRow label="UTC 时间" value={result.UTCTime}/>
          <ResultRow label="相对时间" value={result.RelativeTime}/>
          <ResultRow label="星期" value={result.Weekday}/>
          <ResultRow label="Unix 秒" value={result.UnixSecond}/>
          <ResultRow label="Unix 毫秒" value={result.UnixMilli}/>
        </div>
      )}
    </div>
  );
}

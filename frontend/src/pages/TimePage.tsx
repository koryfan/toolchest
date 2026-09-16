import {useState} from 'react';
import {invoke} from '../actions';
import {useToolContent, useStore} from '../store';
import {showSuccess, showError} from '../components/Notice';
import styles from './TimePage.module.css';

// 时间戳工具页：输入时间戳（秒/毫秒）或日期字符串 → 多格式结果。
// 输入行读写走 store（'time'）；解析走 Invoke('parse')（结果在 Result.Data）；
// 工具栏「当前时间」/「当前时间戳」为 Go 下发按钮，结果 Content 回写输入行（store）。

// 解析结果（Invoke 返回的 Result.Data 字段）
interface TimeData {
  localTime?: string;
  utcTime?: string;
  relativeTime?: string;
  weekday?: string;
  unixSecond?: string;
  unixMilli?: string;
}

// 结果行：label + 值 + 点击复制
function ResultRow({label, value}: { label: string; value: string }) {
  if (!value) return null;
  return (
    <div className={styles.row} onClick={() => {
      navigator.clipboard.writeText(value).then(
        () => showSuccess(`已复制: ${value.length > 40 ? value.slice(0, 40) + '…' : value}`),
        () => showError('复制失败'));
    }}>
      <span className={styles.label}>{label}</span>
      <span className={styles.value}>{value}</span>
    </div>
  );
}

export default function TimePage() {
  const input = useToolContent('time');
  const setContent = useStore(s => s.setContent);
  const replaceContent = useStore(s => s.replaceContent);
  const [result, setResult] = useState<TimeData | null>(null);

  // 回车或点「解析」触发；「当前时间」/「当前时间戳」只填充输入行，解析仍由用户触发
  const parse = (text: string) => {
    if (!text.trim()) { setResult(null); showError('请输入时间戳或日期'); return; }
    void invoke('time', 'parse').then(r => {
      setResult(r.isError ? null : (r.data as TimeData) ?? null);
    });
  };

  // 清空：回写空内容（立即同步 Go 镜像）+ 清掉解析结果，恢复默认形态
  const clear = () => {
    replaceContent('time', '');
    setResult(null);
  };

  return (
    <div className={styles.page}>
      <div className={styles.inputBar}>
        <input
          className={styles.input}
          value={input}
          placeholder="秒/毫秒时间戳或日期字符串，如 1720000000、1720000000000、2024-07-03 15:30:00"
          onChange={e => setContent('time', e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') parse(input); }}
          spellCheck={false}
        />
        <button className={styles.btn} onClick={() => parse(input)}>解析</button>
        <button className={styles.btnSecondary} onClick={clear}>清空</button>
      </div>
      {result && (
        <div className={styles.results}>
          <ResultRow label="本地时间" value={result.localTime ?? ''}/>
          <ResultRow label="UTC 时间" value={result.utcTime ?? ''}/>
          <ResultRow label="相对时间" value={result.relativeTime ?? ''}/>
          <ResultRow label="星期" value={result.weekday ?? ''}/>
          <ResultRow label="Unix 秒" value={result.unixSecond ?? ''}/>
          <ResultRow label="Unix 毫秒" value={result.unixMilli ?? ''}/>
        </div>
      )}
    </div>
  );
}

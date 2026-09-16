import {useEffect} from 'react';
import {invoke} from '../actions';
import {useToolContent, useStore} from '../store';
import {showSuccess, showError} from '../components/Notice';
import styles from './UuidPage.module.css';

// UUID 工具页：展示生成结果（每行一个），点击单个复制；
// 版本/数量选择与生成按钮在工具栏（Go 下发 radio/button）。
// 首次打开内容为空时自动生成一次。

export default function UuidPage() {
  const content = useToolContent('uuid');

  // 首次进入自动生成（静默：不弹「已生成」提示，提示只属于点「生成」按钮）
  useEffect(() => {
    if (!(useStore.getState().tools['uuid']?.content ?? '').trim()) {
      void invoke('uuid', 'gen', '', true);
    }
  }, []);

  const lines = content.split('\n').filter(l => l.trim() !== '');

  return (
    <div className={styles.page}>
      {lines.length === 0 ? (
        <div className={styles.empty}>点击工具栏「生成」按钮</div>
      ) : (
        <div className={styles.list}>
          {lines.map((s, i) => (
            <div
              key={`${i}-${s}`}
              className={styles.row}
              onClick={() => {
                navigator.clipboard.writeText(s).then(
                  () => showSuccess(`已复制: ${s}`),
                  () => showError('复制失败'));
              }}
            >
              <span className={styles.index}>{i + 1}</span>
              <span className={styles.value}>{s}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

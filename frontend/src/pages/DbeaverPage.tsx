import {invoke} from '../actions';
import {useStore} from '../store';
import {showSuccess, showError} from '../components/Notice';
import {Dialogs} from '@wailsio/runtime';
import styles from './DbeaverPage.module.css';

// DBeaver 凭据页：数据由 Go 解密下发（Invoke 结果 Data），
// 本页只渲染 + 点击复制；目录选择/缓存/刷新在工具栏（Go 驱动 Actions）。

interface CredentialEntry {
  id: string;
  name: string;
  driver: string;
  host: string;
  keys: Record<string, string>;
}

/** 复制单值（40 字符以上只提示已复制，与 JSON 树一致） */
const copyValue = (key: string, value: string) => {
  navigator.clipboard.writeText(value).then(
    () => {
      if ([...value].length > 40) {
        showSuccess(`已复制 ${key}`);
      } else {
        showSuccess(`已复制 ${key}: ${value}`);
      }
    },
    () => showError('复制失败'),
  );
};

function Card({entry}: { entry: CredentialEntry }) {
  const keys = Object.keys(entry.keys).sort();
  const title = entry.name || entry.id;
  return (
    <div className={styles.card}>
      <div className={styles.cardTitle}>
        <span>{title}</span>
        {entry.driver && <span className={styles.driver}>{entry.driver}</span>}
        {entry.host && <span className={styles.host}>{entry.host}</span>}
      </div>
      {entry.name && <div className={styles.subId}>{entry.id}</div>}
      {keys.map(k => (
        <div key={k} className={styles.row} onClick={() => copyValue(k, entry.keys[k])}>
          <span className={styles.key}>{k}</span>
          <span className={`${styles.value} ${k === 'password' ? styles.valuePassword : ''}`}>
            {entry.keys[k]}
          </span>
        </div>
      ))}
    </div>
  );
}

export default function DbeaverPage() {
  const entries = useStore(s => s.dbeaverEntries);

  if (!entries) {
    return (
      <div className={styles.page}>
        <div className={styles.hint}>
          点右上「选择目录」，选到 DBeaver 工作目录（含 .dbeaver/ 子目录的那层，如 General），
          解密出的全部连接凭据会列在这里，点击任意行复制。
        </div>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <div className={styles.list}>
        {entries.map(e => <Card key={e.id} entry={e}/>)}
      </div>
    </div>
  );
}

/** 工具栏「选择目录」本地 handler：原生目录选择 → Invoke select */
export async function pickDbeaverDir(): Promise<void> {
  const dirs = await Dialogs.OpenFile({
    Title: '选择 DBeaver 工作目录（如 General）',
    CanChooseDirectories: true,
    CanChooseFiles: false,
    ResolvesAliases: true,
  });
  const dir = Array.isArray(dirs) ? dirs[0] : dirs;
  if (!dir) return; // 取消
  await invoke('dbeaver', 'select', dir);
}

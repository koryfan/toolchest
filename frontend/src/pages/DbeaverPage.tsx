import {invoke} from '../actions';
import {useStore, onToolData, type DbeaverEntry} from '../store';
import {showSuccess, showError} from '../components/Notice';
import {Dialogs} from '@wailsio/runtime';
import styles from './DbeaverPage.module.css';

// DBeaver 凭据页：数据由 Go 解密下发（Invoke 结果 Data），
// 本页只渲染 + 点击复制；目录选择/缓存/刷新在工具栏（Go 驱动 Actions）。


// 数据入口注册：Invoke 结果（select/refresh/export）与事件推送共用此回调。
// 失败结果不带 entries，保留旧列表供对照
onToolData('dbeaver', data => {
  const d = data as {entries?: DbeaverEntry[]; dir?: string};
  if (d.entries) {
    useStore.getState().setDbeaverEntries(d.entries, d.dir);
  }
});
// silent：无缓存目录时 Go 返回「尚未选择目录」是正常启动态，不应弹错
void invoke('dbeaver', 'refresh', '', true).catch(() => {
  // 请求本身失败（桥接异常）也静默，用户可手动刷新
});


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

/** 单条导出文本：名称注释行 + DSN 行（与 Go ExportText 同格式） */
const entryText = (entry: DbeaverEntry): string => `# ${entry.name || entry.id}\n${entry.dsn}`;

function Card({entry}: { entry: DbeaverEntry }) {
  // user 在上、password 随后，其余键字母序
  const order = ['user', 'password'];
  const keys = Object.keys(entry.keys).sort((a, b) => {
    const ia = order.indexOf(a);
    const ib = order.indexOf(b);
    if (ia !== -1 || ib !== -1) return (ia === -1 ? 99 : ia) - (ib === -1 ? 99 : ib);
    return a.localeCompare(b);
  });
  const title = entry.name || entry.id;
  return (
    <div className={styles.card}>
      <div className={styles.cardTitle}>
        <span>{title}</span>
        {entry.driver && <span className={styles.driver}>{entry.driver}</span>}
        {entry.host && <span className={styles.host}>{entry.port ? `${entry.host}:${entry.port}` : entry.host}</span>}
        {entry.database && <span className={styles.host}>{entry.database}</span>}
        {entry.dsn && (
          <button
            className={styles.dsnBtn}
            onClick={() => copyValue('DSN', entryText(entry))}
          >
            复制 DSN
          </button>
        )}
      </div>
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

/** 工具栏「导出」本地 handler：原生保存对话框 → Invoke export（Go 写纯文本 DSN，0600） */
export async function exportDbeaver(): Promise<void> {
  const {dbeaverEntries: entries, dbeaverDir} = useStore.getState();
  if (!entries?.length) {
    showError('尚未打开目录，无以导出');
    return;
  }
  // 默认文件名 = 当前工作目录 basename，如 .../ktv-new-test → ktv-new-test.txt
  const base = dbeaverDir?.split('/').filter(Boolean).pop() || 'dbeaver-dsn';
  const path = await Dialogs.SaveFile({
    Title: '导出 DBeaver 连接 DSN',
    Filename: `${base}.txt`,
    CanCreateDirectories: true,
    Filters: [{DisplayName: '文本文件', Pattern: '*.txt'}],
  });
  if (!path) return; // 取消
  await invoke('dbeaver', 'export', path);
}

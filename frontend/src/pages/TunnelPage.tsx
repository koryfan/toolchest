import {useState} from 'react';
import {invoke} from '../actions';
import {useStore, type TunnelView} from '../store';
import TunnelDialog from './TunnelDialog';
import styles from './TunnelPage.module.css';

// SSH 隧道页：卡片列表（运行中/异常置顶分组）+ 新增卡片 + 编辑弹窗。
// 数据全部来自 Go（tunnel-update 事件推送 + Invoke 结果回写 store），本页只渲染。

const authLabels: Record<string, string> = {
  password: '密码',
  agent: '代理',
  publickey: '公钥',
};

function Card({tun}: { tun: TunnelView }) {
  const [confirmDel, setConfirmDel] = useState(false);

  const op = (opName: string, arg: string) => void invoke('tunnel', opName, arg);

  const cardClass = tun.status === 'running' ? styles.cardRunning
    : tun.status === 'error' ? styles.cardError : '';

  return (
    <div className={`${styles.card} ${cardClass}`}>
      <div className={styles.head}>
        <div className={styles.name}>
          <span
            className={`${styles.statusDot} ${
              tun.status === 'running' ? styles.dotRunning
                : tun.status === 'error' ? styles.dotError : styles.dotStopped
            }`}
          />
          {tun.name}
        </div>
        <span className={styles.auth}>
          {authLabels[tun.authMethod] ?? tun.authMethod}
          {tun.authMethod === 'publickey' && tun.keyPath ? ` · ${tun.keyPath.split('/').pop()}` : ''}
        </span>
      </div>

      <div className={styles.info}>
        <div>{tun.username}@{tun.host}:{tun.port}</div>
        <div className={styles.forward}>
          {tun.localHost}:{tun.localPort} → {tun.remoteHost}:{tun.remotePort}
        </div>
      </div>

      {tun.status === 'error' && tun.errMsg && <div className={styles.errMsg}>{tun.errMsg}</div>}

      <div className={styles.ops}>
        {tun.status === 'running' ? (
          <button onClick={() => op('stop', tun.name)}>停止</button>
        ) : (
          <button onClick={() => op('start', tun.name)}>开启</button>
        )}
        <button onClick={() => op('test', tun.name)}>测试</button>
        <button onClick={() => useStore.getState().openTunnelDialog(tun)}>修改</button>
        <span className={styles.spacer}/>
        {confirmDel ? (
          <>
            <span style={{fontSize: 12, alignSelf: 'center'}}>确认删除？</span>
            <button
              className={styles.danger}
              onClick={() => {
                setConfirmDel(false);
                op('delete', tun.name);
              }}
            >
              是
            </button>
            <button onClick={() => setConfirmDel(false)}>否</button>
          </>
        ) : (
          <button className={styles.danger} onClick={() => setConfirmDel(true)}>删除</button>
        )}
      </div>
    </div>
  );
}

export default function TunnelPage() {
  const data = useStore(s => s.tunnelData);

  const running = data.tunnels.filter(t => t.status !== 'stopped');
  const stopped = data.tunnels.filter(t => t.status === 'stopped');

  return (
    <div className={styles.page}>
      {running.length > 0 && (
        <>
          <div className={styles.groupTitle}>已开启 / 异常（{running.length}）</div>
          <div className={styles.grid}>
            {running.map(t => <Card key={t.name} tun={t}/>)}
          </div>
        </>
      )}
      <div className={styles.groupTitle}>未开启（{stopped.length}）</div>
      {stopped.length > 0 ? (
        <div className={styles.grid}>
          {stopped.map(t => <Card key={t.name} tun={t}/>)}
          <button className={styles.addCard} onClick={() => useStore.getState().openTunnelDialog(null)}>
            + 新增隧道
          </button>
        </div>
      ) : (
        <>
          {data.tunnels.length === 0 && (
            <div className={styles.empty}>还没有隧道配置，点下方新增一个：</div>
          )}
          <div className={styles.grid}>
            <button className={styles.addCard} onClick={() => useStore.getState().openTunnelDialog(null)}>
              + 新增隧道
            </button>
          </div>
        </>
      )}
      <TunnelDialog/>
    </div>
  );
}

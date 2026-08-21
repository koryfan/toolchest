import {useEffect, useRef, useState} from 'react';
import {Invoke} from '../../bindings/toolchest/internal/services/appservice';
import {useStore, type TunnelView} from '../store';
import styles from './TunnelPage.module.css';

// 隧道新增/编辑弹窗：受控表单 + 测试（未保存的表单值直接验证）+ 保存。
// 关闭时只清 UI 状态；保存成功（Go 返回新列表）后自动关闭。

interface FormState {
  name: string;
  host: string;
  port: string;
  username: string;
  authMethod: TunnelView['authMethod'];
  password: string;
  keyPath: string;
  localHost: string;
  localPort: string;
  remoteHost: string;
  remotePort: string;
}

const emptyForm: FormState = {
  name: '', host: '', port: '22', username: '', authMethod: 'agent',
  password: '', keyPath: '', localHost: '127.0.0.1', localPort: '',
  remoteHost: 'localhost', remotePort: '',
};

function fromTunnel(t: TunnelView): FormState {
  return {
    name: t.name, host: t.host, port: String(t.port), username: t.username,
    authMethod: t.authMethod, password: t.password ?? '', keyPath: t.keyPath ?? '',
    localHost: t.localHost, localPort: String(t.localPort),
    remoteHost: t.remoteHost, remotePort: String(t.remotePort),
  };
}

function toConfigJSON(f: FormState): string {
  return JSON.stringify({
    name: f.name.trim(),
    host: f.host.trim(),
    port: Number(f.port) || 0,
    username: f.username.trim(),
    authMethod: f.authMethod,
    password: f.password,
    keyPath: f.keyPath.trim(),
    localHost: f.localHost.trim(),
    localPort: Number(f.localPort) || 0,
    remoteHost: f.remoteHost.trim(),
    remotePort: Number(f.remotePort) || 0,
  });
}

// 认证方式自绘下拉（原生 select 是系统默认样式，与深色主题不搭；
// 视觉复用弹出菜单的深色卡片样式，同工具栏「设置 ▾」一致）
const authOptions: { value: TunnelView['authMethod']; label: string }[] = [
  {value: 'agent', label: '代理（ssh-agent / 默认密钥）'},
  {value: 'publickey', label: '公钥（IdentityFile）'},
  {value: 'password', label: '密码'},
];

function AuthSelect({value, onChange}: {
  value: TunnelView['authMethod'];
  onChange: (v: TunnelView['authMethod']) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, [open]);

  const cur = authOptions.find(o => o.value === value);

  return (
    <div className={styles.authSelectWrap} ref={ref}>
      <div
        className={styles.authSelect}
        onClick={() => setOpen(o => !o)}
      >
        <span>{cur?.label ?? value}</span>
        <span className={styles.authSelectArrow}>▾</span>
      </div>
      {open && (
        <div className={styles.authMenu}>
          {authOptions.map(o => (
            <div
              key={o.value}
              className={o.value === value ? styles.authMenuSelected : undefined}
              onClick={() => {
                setOpen(false);
                onChange(o.value);
              }}
            >
              {o.label}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function TunnelDialog() {
  const dialog = useStore(s => s.tunnelDialog);
  const close = useStore(s => s.closeTunnelDialog);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  // 测试结果（null = 未测）：ok=false 时 msg 为失败原因
  const [testRet, setTestRet] = useState<{ ok: boolean; msg: string } | null>(null);

  // 打开时用编辑目标（或空表单）初始化
  useEffect(() => {
    if (dialog) {
      setForm(dialog.editing ? fromTunnel(dialog.editing) : emptyForm);
      setTestRet(null);
    }
  }, [dialog]);

  if (!dialog) return null;
  const editing = dialog.editing !== null;

  const set = (patch: Partial<FormState>) => setForm(f => ({...f, ...patch}));

  const onTest = async () => {
    setTesting(true);
    setTestRet(null);
    try {
      const r = await Invoke('tunnel', 'test', toConfigJSON(form));
      setTestRet({ok: !r.isError, msg: r.notice});
    } catch (e) {
      setTestRet({ok: false, msg: String(e)});
    } finally {
      setTesting(false);
    }
  };

  const onSave = async () => {
    setSaving(true);
    try {
      const r = await Invoke('tunnel', 'save', toConfigJSON(form));
      if (!r.isError) {
        // 保存成功：结果里的列表直接刷进 store（invoke 已处理 notice）
        const data = r.data as { tunnels?: TunnelView[] } | null;
        if (data?.tunnels) {
          useStore.getState().setTunnelData({
            tunnels: data.tunnels,
            running: data.tunnels.filter(t => t.status === 'running').length,
          });
        }
        close();
      }
    } finally {
      setSaving(false);
    }
  };

  const numInput = (value: string, onChange: (v: string) => void, label: string) => (
    <div className={styles.field}>
      <label>{label}</label>
      <input value={value} onChange={e => onChange(e.target.value)} inputMode="numeric" spellCheck={false}/>
    </div>
  );

  return (
    <div className={styles.overlay} onMouseDown={e => { if (e.target === e.currentTarget) close(); }}>
      <div className={styles.dialog}>
        <h2 className={styles.dialogTitle}>{editing ? `编辑隧道 · ${form.name}` : '新增隧道'}</h2>

        <div className={styles.formGrid}>
          <div className={styles.field}>
            <label>隧道名称</label>
            <input
              value={form.name}
              disabled={editing} // 名称是唯一标识，编辑态不改
              onChange={e => set({name: e.target.value})}
              spellCheck={false}
            />
          </div>
          <div className={styles.field}>
            <label>主机 / IP</label>
            <input value={form.host} onChange={e => set({host: e.target.value})} spellCheck={false}/>
          </div>
          {numInput(form.port, v => set({port: v}), 'SSH 端口')}
          <div className={styles.field}>
            <label>用户名</label>
            <input value={form.username} onChange={e => set({username: e.target.value})} spellCheck={false}/>
          </div>

          <div className={styles.field}>
            <label>认证方式</label>
            <AuthSelect
              value={form.authMethod}
              onChange={v => set({authMethod: v})}
            />
          </div>
          {form.authMethod === 'password' && (
            <div className={styles.field}>
              <label>密码</label>
              <input
                type="password"
                value={form.password}
                onChange={e => set({password: e.target.value})}
                spellCheck={false}
              />
            </div>
          )}
          {form.authMethod === 'publickey' && (
            <div className={styles.field}>
              <label>密钥路径（支持 ~）</label>
              <input
                value={form.keyPath}
                placeholder="~/.ssh/id_rsa"
                onChange={e => set({keyPath: e.target.value})}
                spellCheck={false}
              />
            </div>
          )}

          <div className={styles.sectionTitle}>端口转发（ssh -L 等价）</div>
          <div className={styles.field}>
            <label>本地主机</label>
            <input value={form.localHost} onChange={e => set({localHost: e.target.value})} spellCheck={false}/>
          </div>
          {numInput(form.localPort, v => set({localPort: v}), '本地端口')}
          <div className={styles.field}>
            <label>远程主机（SSH 服务端视角）</label>
            <input value={form.remoteHost} onChange={e => set({remoteHost: e.target.value})} spellCheck={false}/>
          </div>
          {numInput(form.remotePort, v => set({remotePort: v}), '远程端口')}

          {testRet && (
            <div className={`${styles.testResult} ${testRet.ok ? styles.testOk : styles.testErr}`}>
              {testRet.msg}
            </div>
          )}
        </div>

        <div className={styles.dialogOps}>
          <button onClick={onTest} disabled={testing}>{testing ? '测试中…' : '测试隧道配置'}</button>
          <button className={styles.primary} onClick={onSave} disabled={saving}>
            {saving ? '保存中…' : '保存隧道配置'}
          </button>
        </div>
      </div>
    </div>
  );
}

// 内容镜像 store：Go 是内容权威，前端本地即时更新保证输入流畅，
// debounce 300ms 推 UpdateContent 同步镜像；op 结果经 replaceContent 即时回写。
// 隧道数据由 Go 事件（tunnel-update）推送 + 初始 list 拉取，前端只渲染。
import {create} from 'zustand';
import {Events} from '@wailsio/runtime';
import {UpdateContent, Invoke} from '../bindings/toolchest/internal/services/appservice';
import type {Action, ToolDesc} from '../bindings/toolchest/internal/domain/models';

/** 隧道视图（Go tunneltool.TunnelView 经 JSON 序列化） */
export interface TunnelView {
  name: string;
  host: string;
  port: number;
  username: string;
  authMethod: 'password' | 'agent' | 'publickey';
  password?: string;
  keyPath?: string;
  localHost: string;
  localPort: number;
  remoteHost: string;
  remotePort: number;
  status: 'stopped' | 'running' | 'error';
  errMsg?: string;
}

export interface TunnelData {
  tunnels: TunnelView[];
  running: number;
}

/** DBeaver 凭据条目（Go dbeavertool.CredentialEntry） */
export interface DbeaverEntry {
  id: string;
  name: string;
  driver: string;
  host: string;
  keys: Record<string, string>;
}

const DEBOUNCE_MS = 300;

interface ToolState {
  content: string;
}

interface AppState {
  /** 各工具内容镜像（内存，重启即空——内容不入盘） */
  tools: Record<string, ToolState>;
  /** 激活工具 ID */
  activeId: string;
  /** 各工具的操作区配置（Tools() 下发 + Invoke 回传刷新） */
  actions: Record<string, Action[]>;
  /** 工具元信息列表（Tools() 下发，侧栏渲染依据） */
  descs: ToolDesc[];
  /** 隧道列表（事件推送 + 初始 list） */
  tunnelData: TunnelData;
  /** 隧道编辑弹窗（null = 关闭） */
  tunnelDialog: {editing: TunnelView | null} | null;
  /** DBeaver 凭据列表（null = 尚未打开过目录） */
  dbeaverEntries: DbeaverEntry[] | null;
  setContent: (toolID: string, content: string) => void;
  setTunnelData: (d: TunnelData) => void;
  openTunnelDialog: (editing: TunnelView | null) => void;
  closeTunnelDialog: () => void;
  setDbeaverEntries: (entries: DbeaverEntry[] | null) => void;
  replaceContent: (toolID: string, content: string) => void;
  flush: (toolID: string) => void;
  setActions: (toolID: string, actions: Action[]) => void;
  setDescs: (descs: ToolDesc[]) => void;
  setActiveId: (id: string) => void;
}

// debounce 定时器表（模块级，不在 store 里——无渲染意义）
const timers: Record<string, ReturnType<typeof setTimeout> | undefined> = {};
const pending: Record<string, string | undefined> = {};

function push(toolID: string, content: string) {
  void UpdateContent(toolID, content).catch(() => {
    // 镜像同步 fire-and-forget：失败静默，允许短暂落后
  });
}

export const useStore = create<AppState>((set, get) => ({
  tools: {},
  activeId: 'json',
  actions: {},
  descs: [],
  tunnelData: {tunnels: [], running: 0},
  tunnelDialog: null,
  dbeaverEntries: null,

  setContent: (toolID, content) => {
    set(s => ({
      tools: {...s.tools, [toolID]: {content}},
    }));
    pending[toolID] = content;
    clearTimeout(timers[toolID]);
    timers[toolID] = setTimeout(() => {
      const c = pending[toolID];
      if (c !== undefined) {
        delete pending[toolID];
        push(toolID, c);
      }
    }, DEBOUNCE_MS);
  },

  replaceContent: (toolID, content) => {
    // op 结果回写：取消 pending 的镜像推送，立即同步（内容已是权威值）
    clearTimeout(timers[toolID]);
    delete pending[toolID];
    set(s => ({
      tools: {...s.tools, [toolID]: {content}},
    }));
    push(toolID, content);
  },

  flush: toolID => {
    const c = pending[toolID];
    clearTimeout(timers[toolID]);
    if (c !== undefined) {
      delete pending[toolID];
      push(toolID, c);
    }
  },

  setActions: (toolID, actions) => {
    set(s => ({actions: {...s.actions, [toolID]: actions}}));
  },

  setDescs: descs => {
    set({descs});
  },

  setTunnelData: d => {
    set({tunnelData: d});
  },

  openTunnelDialog: editing => {
    set({tunnelDialog: {editing}});
  },

  closeTunnelDialog: () => {
    set({tunnelDialog: null});
  },

  setDbeaverEntries: entries => {
    set({dbeaverEntries: entries});
  },

  setActiveId: id => {
    set({activeId: id});
  },
}));

/** 订阅某工具的内容（页面编辑器 value 数据源） */
export function useToolContent(toolID: string): string {
  return useStore(s => s.tools[toolID]?.content ?? '');
}

// ---- 隧道事件订阅（模块级初始化一次） ----

let tunnelEventsBound = false;

export function initTunnelEvents(): void {
  if (tunnelEventsBound) return;
  tunnelEventsBound = true;
  Events.On('tunnel-update', ev => {
    const data = ev.data as unknown as TunnelData;
    if (data && Array.isArray(data.tunnels)) {
      useStore.getState().setTunnelData(data);
    }
  });
  // 初始拉取一次（事件只在状态变化时发）
  void invokeTunnelList();
}

// DBeaver：启动时若有缓存目录（Go State.CurrentDir），refresh 恢复列表
export function initDbeaverRestore(): void {
  void Invoke('dbeaver', 'refresh', '').catch(() => {
    // 无缓存目录时 Go 返回错误（尚未选择目录），静默忽略
  });
}

async function invokeTunnelList(): Promise<void> {
  const r = await Invoke('tunnel', 'list', '');
  const data = r.data as unknown as TunnelData;
  if (data && Array.isArray(data.tunnels)) {
    useStore.getState().setTunnelData(data);
  }
}

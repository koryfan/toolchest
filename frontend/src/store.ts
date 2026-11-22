// 内容镜像 store：Go 是内容权威，前端本地即时更新保证输入流畅，
// debounce 300ms 推 UpdateContent 同步镜像；op 结果经 replaceContent 即时回写。
// 后台数据由 Go 事件（tool-update）推送，经各页面注册的回调分发，
// 本文件只做通用镜像/描述/弹窗状态，不认识任何具体工具。
import {create} from 'zustand';
import {Events} from '@wailsio/runtime';
import {UpdateContent, Tools} from '../bindings/toolchest/internal/services/appservice';
import type {Action, ToolDesc} from '../bindings/toolchest/internal/domain/models';

/** 隧道视图（Go tunneltool.TunnelView 经 JSON 序列化） */
export interface TunnelView {
  name: string;
  host: string;
  port: number;
  username: string;
  authMethod: 'password' | 'agent' | 'publickey';
  forwardMode?: 'direct' | 'exec';
  execCommand?: string;
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
  port: number;
  database: string;
  dsn: string;
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
  /** 隧道编辑弹窗（null = 关闭）；editing = 修改目标，seed = 复制来源（新增态预填，名称清空） */
  tunnelDialog: {editing: TunnelView | null; seed?: TunnelView} | null;
  /** DBeaver 凭据列表（null = 尚未打开过目录） */
  dbeaverEntries: DbeaverEntry[] | null;
  /** DBeaver 当前工作目录（导出默认文件名取 basename） */
  dbeaverDir: string | null;
  setContent: (toolID: string, content: string) => void;
  setTunnelData: (d: TunnelData) => void;
  copyTunnel: (t: TunnelView) => void;
  openTunnelDialog: (editing: TunnelView | null) => void;
  closeTunnelDialog: () => void;
  setDbeaverEntries: (entries: DbeaverEntry[] | null, dir?: string) => void;
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
  dbeaverDir: null,

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

  copyTunnel: t => {
    set({tunnelDialog: {editing: null, seed: t}});
  },

  closeTunnelDialog: () => {
    set({tunnelDialog: null});
  },

  setDbeaverEntries: (entries, dir) => {
    set({dbeaverEntries: entries, dbeaverDir: dir ?? null});
  },

  setActiveId: id => {
    set({activeId: id});
  },
}));

/** 订阅某工具的内容（页面编辑器 value 数据源） */
export function useToolContent(toolID: string): string {
  return useStore(s => s.tools[toolID]?.content ?? '');
}

// ---- 工具后台数据分发（模块级初始化一次） ----

// 各工具的数据消费回调（页面模块加载时注册）：Go 事件推送与 Invoke 结果
// 两条入口共用一张表，通用层不认识任何具体工具
const dataHandlers: Record<string, (data: any) => void> = {};

/** 注册某工具的数据消费回调（页面模块顶层调用一次） */
export function onToolData(toolID: string, handler: (data: any) => void): void {
  dataHandlers[toolID] = handler;
}

/** 分发一份工具数据（Invoke 结果 / tool-update 事件共用） */
export function dispatchToolData(toolID: string, data: unknown): void {
  dataHandlers[toolID]?.(data);
}

/** 拉取工具描述：actions/descs 刷新 + activeId 失效回退首个 */
export async function refreshDescs(): Promise<void> {
  const list = await Tools();
  if (!list || list.length === 0) return;
  const store = useStore.getState();
  store.setDescs(list);
  for (const d of list) {
    store.setActions(d.meta.id, d.actions ?? []);
  }
  if (!list.some(d => d.meta.id === store.activeId)) {
    store.setActiveId(list[0].meta.id);
  }
}

let toolEventsBound = false;

/** 订阅 tool-update 事件（Go 后台状态变化 → 分发数据 + 刷新侧栏提醒灯） */
export function initToolEvents(): void {
  if (toolEventsBound) return;
  toolEventsBound = true;
  Events.On('tool-update', ev => {
    const payload = ev.data as unknown as {id?: string; data?: unknown};
    if (!payload?.id) return;
    dispatchToolData(payload.id, payload.data);
    // 提醒灯（ToolDesc.Running）随后台状态变化刷新
    void refreshDescs().catch(() => {
      // 刷新失败静默，下次事件再试
    });
  });
}

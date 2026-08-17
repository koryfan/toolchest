// 内容镜像 store：Go 是内容权威，前端本地即时更新保证输入流畅，
// debounce 300ms 推 UpdateContent 同步镜像；op 结果经 replaceContent 即时回写。
import {create} from 'zustand';
import {UpdateContent} from '../bindings/tool-dev/internal/services/appservice';
import type {Action, ToolDesc} from '../bindings/tool-dev/internal/domain/models';

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
  setContent: (toolID: string, content: string) => void;
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

  setActiveId: id => {
    set({activeId: id});
  },
}));

/** 订阅某工具的内容（页面编辑器 value 数据源） */
export function useToolContent(toolID: string): string {
  return useStore(s => s.tools[toolID]?.content ?? '');
}

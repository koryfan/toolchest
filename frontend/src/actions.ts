// 通用操作分发：所有工具操作走 AppService.Invoke；
// Action.Local 命中的前端本地操作（清空/复制）走本文件注册表。
import {Invoke} from '../bindings/tool-dev/internal/services/appservice';
import type {Action, Result} from '../bindings/tool-dev/internal/domain/models';
import {useStore} from './store';
import {showSuccess, showError} from './components/Notice';

/** 本地操作上下文 */
interface LocalCtx {
  toolID: string;
}

type LocalHandler = (ctx: LocalCtx) => void;

/**
 * 前端本地 handler 注册表：Action.Local 的值即这里的键。
 * Go 配置删掉 Local 字段（或改名）后前端自动无此操作。
 */
const localHandlers: Record<string, LocalHandler> = {
  clear: ({toolID}) => {
    useStore.getState().replaceContent(toolID, '');
  },
  copy: ({toolID}) => {
    const content = useStore.getState().tools[toolID]?.content ?? '';
    const n = [...content].length;
    if (n === 0) {
      showError('内容为空，无以复制');
      return;
    }
    navigator.clipboard.writeText(content).then(
      () => showSuccess(`已复制全部内容（${n} 字符）`),
      () => showError('复制失败'),
    );
  },
};

/** 执行一个操作并统一处理 Result（内容回写/提示/配置刷新） */
export async function invoke(toolID: string, op: string, arg = ''): Promise<Result> {
  const r = await Invoke(toolID, op, arg);
  const store = useStore.getState();
  if (r.content) {
    store.replaceContent(toolID, r.content);
  }
  if (r.notice) {
    if (r.isError) {
      showError(r.notice);
    } else {
      showSuccess(r.notice);
    }
  }
  if (r.actions) {
    store.setActions(toolID, r.actions);
  }
  return r;
}

/** 点击一个 Action：本地操作走注册表，否则 Invoke */
export function runAction(toolID: string, action: Action, arg = ''): void {
  if (action.local) {
    localHandlers[action.local]?.({toolID});
    return;
  }
  void invoke(toolID, action.id, arg);
}

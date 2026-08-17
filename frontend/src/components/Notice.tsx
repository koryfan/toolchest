import {createContext, useContext, type ReactNode} from 'react';
import toast, {Toaster} from 'react-hot-toast';

// 全局提示：基于 react-hot-toast（toast 本身是全局单例）。
// 导出纯函数 showSuccess/showError 供非 React 上下文（actions.ts 等）调用；
// useNotice() hook 保留给组件内调用点，两者同一实现。
// 样式用 react-hot-toast 原生默认样式（白底胶囊 + 图标），不做深色定制。

interface NoticeApi {
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

/** 非 React 上下文调用（store/actions 等模块） */
export const showSuccess = (msg: string) => toast.success(msg, {duration: 3000});
export const showError = (msg: string) => toast.error(msg, {duration: 5000});

const api: NoticeApi = {showSuccess, showError};
const NoticeContext = createContext<NoticeApi | null>(null);

export function useNotice(): NoticeApi {
  const ctx = useContext(NoticeContext);
  if (!ctx) throw new Error('useNotice 必须在 NoticeProvider 内使用');
  return ctx;
}

export function NoticeProvider({children}: {children: ReactNode}) {
  return (
    <NoticeContext.Provider value={api}>
      {children}
      <Toaster position="top-center" gutter={8}/>
    </NoticeContext.Provider>
  );
}

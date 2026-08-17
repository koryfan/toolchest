import {createContext, useContext, type ReactNode} from 'react';
import toast, {Toaster} from 'react-hot-toast';

// 全局提示：基于 react-hot-toast（成熟 toast 库，自动管理堆叠/动画/定时消失）。
// useNotice() 暴露 showSuccess/showError，各操作调用点不用关心实现。
// 样式复刻定稿的深色主题胶囊（状态色边框 + 深色底）。

interface NoticeApi {
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

const NoticeContext = createContext<NoticeApi | null>(null);

export function useNotice(): NoticeApi {
  const ctx = useContext(NoticeContext);
  if (!ctx) throw new Error('useNotice 必须在 NoticeProvider 内使用');
  return ctx;
}

export function NoticeProvider({children}: { children: ReactNode }) {
  const api: NoticeApi = {
    // 用 react-hot-toast 原生默认样式（白底胶囊 + 图标），不做深色定制
    showSuccess: (msg: string) => toast.success(msg, {duration: 3000}),
    showError: (msg: string) => toast.error(msg, {duration: 5000}),
  };

  return (
    <NoticeContext.Provider value={api}>
      {children}
      <Toaster position="top-center" gutter={8}/>
    </NoticeContext.Provider>
  );
}

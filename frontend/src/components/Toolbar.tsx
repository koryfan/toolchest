import type {ReactNode} from 'react';
import styles from './Toolbar.module.css';

// 工具页顶部工具栏：左侧工具名 + 右侧动态操作区（由各工具页内聚提供），
// 底部 1px 分隔线（对应旧 Fyne 版定稿布局）
interface Props {
  title: string;
  actions?: ReactNode;
}

export default function Toolbar({title, actions}: Props) {
  return (
    <header className={styles.toolbar}>
      <h1 className={styles.title}>{title}</h1>
      <div className="toolbar-actions">{actions}</div>
    </header>
  );
}

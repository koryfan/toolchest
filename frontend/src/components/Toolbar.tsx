import type {Action} from '../../bindings/toolchest/internal/domain/models';
import ActionRender from './ActionRender';
import styles from './Toolbar.module.css';

// 工具页顶部工具栏：左侧工具名 + 右侧操作区（消费 Go 下发的 Action 配置渲染），
// 底部 1px 分隔线（对应旧 Fyne 版定稿布局）
interface Props {
  title: string;
  toolID: string;
  actions: Action[];
}

export default function Toolbar({title, toolID, actions}: Props) {
  return (
    <header className={styles.toolbar}>
      <h1 className={styles.title}>{title}</h1>
      <div className="toolbar-actions">
        {actions.map(a => (
          <ActionRender key={a.id} toolID={toolID} action={a}/>
        ))}
      </div>
    </header>
  );
}

// 通用操作区渲染器：消费 Go 下发的 Action 配置渲染三种形态
// （button 纯文字按钮 / menu 下拉菜单 / radio 单选组），
// 点击统一走 runAction（本地 handler 注册表或 Invoke）。
// 样式复用定稿视觉：.toolbar-actions 全局类 + 弹出菜单深色卡片。
import {useEffect, useRef, useState} from 'react';
import {ActionType, type Action} from '../../bindings/toolchest/internal/domain/models';
import {runAction} from '../actions';
import styles from './ActionRender.module.css';

interface Props {
  toolID: string;
  action: Action;
}

export default function ActionRender({toolID, action}: Props) {
  switch (action.type) {
    case ActionType.ActionButton:
      return (
        <button
          
          onClick={() => runAction(toolID, action)}
        >
          {action.label}
        </button>
      );
    case ActionType.ActionMenu:
      return <MenuAction toolID={toolID} action={action}/>;
    case ActionType.ActionRadio:
      return <RadioAction toolID={toolID} action={action}/>;
    default:
      return null;
  }
}

/** 下拉菜单：点开菜单项列表，点击项触发对应 op */
function MenuAction({toolID, action}: Props) {
  const [open, setOpen] = useState(false);
  const ref = useOutsideClose(open, () => setOpen(false));

  return (
    <div className={styles.menuWrap} ref={ref}>
      <button
        
        onClick={() => setOpen(o => !o)}
      >
        {action.label} ▾
      </button>
      {open && (
        <div className={styles.menu}>
          {(action.items ?? []).map(item => (
            <div
              key={item.id}
              onClick={() => {
                setOpen(false);
                runAction(toolID, {...action, id: item.id, local: ''});
              }}
            >
              {item.label}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/** 单选组：点开选项列表，当前项高亮，选中即触发（Invoke 自身 ID + value） */
function RadioAction({toolID, action}: Props) {
  const [open, setOpen] = useState(false);
  const ref = useOutsideClose(open, () => setOpen(false));

  return (
    <div className={styles.menuWrap} ref={ref}>
      <button
        
        onClick={() => setOpen(o => !o)}
      >
        {action.label} ▾
      </button>
      {open && (
        <div className={styles.menu}>
          {(action.options ?? []).map(opt => (
            <div
              key={opt.value}
              className={opt.value === action.current ? styles.selected : undefined}
              onClick={() => {
                setOpen(false);
                runAction(toolID, action, opt.value);
              }}
            >
              {opt.label}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/** 点击组件外部时关闭弹出层 */
function useOutsideClose(active: boolean, close: () => void) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!active) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        close();
      }
    };
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, [active, close]);
  return ref;
}

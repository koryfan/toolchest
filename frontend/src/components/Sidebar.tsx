import styled from './Sidebar.module.css';

// 左侧极窄图标侧栏（64px，贯通全高）：图标垂直排列，
// 选中项蓝紫圆角方块高亮 + 白图标，未选中灰图标，hover 浅背景
interface Props {
  icons: { key: string; url: string; title: string; selected: boolean; onClick: () => void }[];
}

export default function Sidebar({icons}: Props) {
  return (
    <nav className={styled.sidebar}>
      {icons.map(it => (
        <button
          key={it.key}
          className={`${styled.item} ${it.selected ? styled.selected : ''}`}
          title={it.title}
          onClick={it.onClick}
        >
          <img src={it.url} alt={it.title} width={24} height={24}/>
        </button>
      ))}
    </nav>
  );
}

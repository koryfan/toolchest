import styles from './TimePage.module.css';

// 时间戳工具页（骨架，待实现）：输入框 + 多行结果 + 工具栏「当前时间戳」
export function TimeActions() {
  return (
    <div>
      <button onClick={() => console.log('TODO: 当前时间戳')}>当前时间戳</button>
    </div>
  );
}

export default function TimePage() {
  return (
    <div className={styles.page}>
      <p className={styles.placeholder}>时间戳转换（待实现）</p>
    </div>
  );
}

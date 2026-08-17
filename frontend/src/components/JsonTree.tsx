import {useMemo} from 'react';
import JsonView from '@uiw/react-json-view';
import {useNotice} from './Notice';
import styles from './JsonTree.module.css';

// JSON 树视图：@uiw/react-json-view 封装（折叠/复制/分色开箱即用）。
// 分色与代码视图（CodeMirror jsonHighlight）同一套，保持两个视图观感一致。

interface Props {
  input: string; // 代码视图当前文本
}

export default function JsonTree({input}: Props) {
  const notice = useNotice();

  const {data, err} = useMemo(() => {
    if (!input.trim()) return {data: undefined, err: ''};
    try {
      return {data: JSON.parse(input) as object, err: ''};
    } catch (e) {
      return {data: undefined, err: `JSON 解析失败: ${(e as Error).message}`};
    }
  }, [input]);

  if (!input.trim()) return <div className={styles.placeholder}>粘贴或输入 JSON 后切换到这里查看树形结构</div>;
  if (err) return <div className={styles.placeholder}>{err}</div>;

  return (
    <div className={styles.pane}>
      <JsonView
        value={data}
        collapsed={5}   // 默认展开 5 层
        displayDataTypes={false}
        // 复制内容可能是多行 JSON 片段，塞 toast 会换行撑得很丑——只给单行简短提示
        onCopied={text => notice.showSuccess(text.includes('\n') ? '已复制' : `已复制: ${text.length > 40 ? text.slice(0, 40) + '…' : text}`)}
        style={{
          backgroundColor: 'transparent',
          fontSize: '16px',   // 与代码视图（JsonEditor 16px）一致
          fontFamily: '"Maple Mono", monospace',
          // 分色对齐代码视图：key 浅灰白/字符串绿/数字蓝/布尔橙/null 灰
          '--w-rjv-color': 'rgb(199, 202, 209)',
          '--w-rjv-key-string': 'rgb(199, 202, 209)',
          '--w-rjv-info-color': 'rgb(121, 125, 134)',   // 折叠计数「N items」
          '--w-rjv-curlybraces-color': 'rgb(121, 125, 134)',
          '--w-rjv-brackets-color': 'rgb(121, 125, 134)',
          '--w-rjv-quotes-color': 'rgb(199, 202, 209)',
          '--w-rjv-type-string-color': 'rgb(112, 219, 184)',
          '--w-rjv-type-int-color': 'rgb(103, 166, 228)',
          '--w-rjv-type-float-color': 'rgb(103, 166, 228)',
          '--w-rjv-type-boolean-color': 'rgb(235, 153, 71)',
          '--w-rjv-type-null-color': 'rgb(121, 125, 134)',
        } as React.CSSProperties}
      />
    </div>
  );
}

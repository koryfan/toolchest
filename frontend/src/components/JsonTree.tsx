import {useEffect, useMemo, useState} from 'react';
import JsonView from '@uiw/react-json-view';
import {useNotice} from './Notice';
import styles from './JsonTree.module.css';

// JSON 树视图：@uiw/react-json-view 封装（折叠/复制/分色开箱即用）。
// 分色与代码视图（CodeMirror jsonHighlight）同一套，保持两个视图观感一致。
// 树内搜索：按 key / 基本类型值子串匹配（大小写不敏感），命中节点自动展开
// 祖先链并高亮（<mark>）。纯前端计算，不涉后端；引号等装饰由库原样渲染。

interface Props {
  input: string; // 代码视图当前文本
}

// render 覆盖的入参形态（对应库的 TypesElement / SectionElement render 签名，取宽松超类型；
// as/render 是库自身的定制字段，必须从透传 props 里剔除，否则会漏到 DOM 上）
type ElemProps = React.HTMLAttributes<HTMLSpanElement> & {
  children?: React.ReactNode;
  as?: unknown;
  render?: unknown;
};
type RenderResult = {type?: string; keys?: (string | number)[]};

// keys 路径 → 稳定字符串键（命中集合用）
const pk = (keys: readonly (string | number)[] | undefined) => JSON.stringify(keys ?? []);

interface SearchSets {
  keyHits: Set<string>; // key 命中的节点路径（含该 key 自身）
  valHits: Set<string>; // 基本类型值命中的节点路径
  expand: Set<string>;  // 需展开的容器路径（命中节点及其全部祖先 + 根）
}

// 记一次命中：进对应命中集合，并标记该节点及祖先链为需展开
function markHit(s: SearchSets, path: (string | number)[], kind: 'key' | 'val'): void {
  (kind === 'key' ? s.keyHits : s.valHits).add(pk(path));
  s.expand.add(pk([])); // 根是所有节点的祖先
  for (let i = 1; i <= path.length; i++) s.expand.add(pk(path.slice(0, i)));
}

// 递归收集命中（只走对象/数组，基本类型由父层判定）
function collect(node: unknown, keys: (string | number)[], q: string, s: SearchSets): void {
  if (node === null || typeof node !== 'object') return;
  const entries: [string | number, unknown][] = Array.isArray(node)
    ? node.map((v, i) => [i, v])
    : Object.entries(node);
  for (const [k, v] of entries) {
    const path = [...keys, k];
    if (String(k).toLowerCase().includes(q)) markHit(s, path, 'key');
    if (v !== null && typeof v === 'object') {
      collect(v, path, q, s);
    } else if (String(v).toLowerCase().includes(q)) {
      markHit(s, path, 'val');
    }
  }
}

export default function JsonTree({input}: Props) {
  const notice = useNotice();
  const [raw, setRaw] = useState('');     // 搜索框输入
  const [query, setQuery] = useState(''); // debounce 250ms 后生效的查询词

  useEffect(() => {
    const t = setTimeout(() => setQuery(raw.trim()), 250);
    return () => clearTimeout(t);
  }, [raw]);

  const {data, err} = useMemo(() => {
    if (!input.trim()) return {data: undefined, err: ''};
    try {
      return {data: JSON.parse(input) as object, err: ''};
    } catch (e) {
      return {data: undefined, err: `JSON 解析失败: ${(e as Error).message}`};
    }
  }, [input]);

  const s = useMemo(() => {
    const q = query.toLowerCase();
    if (!q || data === undefined) return null;
    const sets: SearchSets = {keyHits: new Set(), valHits: new Set(), expand: new Set()};
    collect(data, [], q, sets);
    return {q, ...sets};
  }, [data, query]);

  // 查询词变化即重挂载：展开状态重算（命中链展开、无关分支折叠），丢弃手动展开残留
  const viewKey = 'tree:' + query;

  // 值高亮 render：命中集合不同（key/val），逻辑一致——未命中返回 undefined 走库默认
  const highlight = (hits: Set<string>) => (props: ElemProps, r: RenderResult): React.ReactNode => {
    if (!s || r.type === 'type') return;
    if (!hits.has(pk(r.keys))) return;
    const text = String(props.children);
    const i = text.toLowerCase().indexOf(s.q);
    if (i < 0) return;
    const rest: Record<string, unknown> = {...props};
    delete rest.children;
    delete rest.as;
    delete rest.render;
    return (
      <span {...rest}>
        {text.slice(0, i)}
        <mark className={styles.hit}>{text.slice(i, i + s.q.length)}</mark>
        {text.slice(i + s.q.length)}
      </span>
    );
  };
  const keyRender = highlight(s?.keyHits ?? new Set());
  const valRender = highlight(s?.valHits ?? new Set());

  if (!input.trim()) return <div className={styles.placeholder}>粘贴或输入 JSON 后切换到这里查看树形结构</div>;
  if (err) return <div className={styles.placeholder}>{err}</div>;

  const total = (s?.keyHits.size ?? 0) + (s?.valHits.size ?? 0);

  return (
    <div className={styles.pane}>
      <div className={styles.searchbar}>
        <input
          className={styles.searchInput}
          value={raw}
          placeholder="搜索键或值…"
          onChange={e => setRaw(e.target.value)}
          onKeyDown={e => e.key === 'Escape' && setRaw('')}
        />
        {query && (
          <span className={total ? styles.hitCount : styles.hitCountZero}>
            {total ? `命中 ${total} 处` : '无命中'}
          </span>
        )}
      </div>
      <div className={styles.treeArea}>
        <JsonView
          key={viewKey}
          value={data}
          collapsed={s ? undefined : 5}   // 无搜索时默认展开 5 层；搜索时交给 shouldExpandNodeInitially
          // 实测（SSR 探针）：返回 true = 展开。命中链上的容器展开，其余折叠
          shouldExpandNodeInitially={s ? (_next: boolean, props: {keys?: (string | number)[]}) => s.expand.has(pk(props.keys)) : undefined}
          displayDataTypes={false}
          highlightUpdates={!s}           // 搜索时关掉更新高亮，避免与命中高亮混淆
          shortenTextAfterLength={s ? 0 : undefined}  // 搜索时展示完整长字符串，避免命中被截断隐藏
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
        >
          {s && (
            <>
              <JsonView.KeyName render={keyRender}/>
              <JsonView.String render={valRender}/>
              <JsonView.Int render={valRender}/>
              <JsonView.Float render={valRender}/>
              <JsonView.True render={valRender}/>
              <JsonView.False render={valRender}/>
              <JsonView.Null render={valRender}/>
              <JsonView.Undefined render={valRender}/>
            </>
          )}
        </JsonView>
      </div>
    </div>
  );
}

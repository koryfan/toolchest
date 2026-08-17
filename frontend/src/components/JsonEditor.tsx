import {useEffect, useRef} from 'react';
import {EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter} from '@codemirror/view';
import {EditorState} from '@codemirror/state';
import {defaultKeymap, history, historyKeymap, indentWithTab} from '@codemirror/commands';
import {foldGutter, syntaxHighlighting, HighlightStyle} from '@codemirror/language';
import {search, searchKeymap, SearchQuery, SearchCursor, setSearchQuery, findNext, findPrevious, closeSearchPanel} from '@codemirror/search';
import {autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap} from '@codemirror/autocomplete';
import {lintKeymap} from '@codemirror/lint';
import {json} from '@codemirror/lang-json';
import {tags} from '@lezer/highlight';
import {indentationMarkers} from '@replit/codemirror-indentation-markers';
import type {Panel} from '@codemirror/view';

// CodeMirror 编辑器封装：行号 + 代码折叠 + JSON 语法高亮 + 缩进参考线 +
// VS Code 风格浮动搜索框（右上角圆角小窗：输入框 + 计数 + 上/下 + 关闭）+
// 括号自动闭合。颜色对齐深色主题（JSON Hero 风格分色）。

interface Props {
  value: string;
  onChange: (v: string) => void;
}

// 语法高亮分色对齐主题：key 浅灰白 / 字符串绿 / 数字蓝 / 布尔橙 / null 灰
const jsonHighlight = HighlightStyle.define([
  {tag: tags.propertyName, color: 'rgb(199, 202, 209)'},
  {tag: tags.string, color: 'rgb(112, 219, 184)'},
  {tag: tags.number, color: 'rgb(103, 166, 228)'},
  {tag: tags.bool, color: 'rgb(235, 153, 71)'},
  {tag: tags.null, color: 'rgb(121, 125, 134)'},
]);

// VS Code 风格搜索面板：浮动右上角的小窗
function vscodeSearchPanel(view: EditorView): Panel {
  const dom = document.createElement('div');
  dom.className = 'cm-search-widget';

  const input = document.createElement('input');
  input.placeholder = '查找';
  input.spellcheck = false;

  const count = document.createElement('span');
  count.className = 'cm-search-count';
  count.textContent = '无结果';

  const mkBtn = (label: string, title: string, onClick: () => void) => {
    const b = document.createElement('button');
    b.textContent = label;
    b.title = title;
    b.onclick = onClick;
    return b;
  };
  const prev = mkBtn('↑', '上一个 (Shift+Enter)', () => findPrevious(view));
  const next = mkBtn('↓', '下一个 (Enter)', () => findNext(view));
  const close = mkBtn('✕', '关闭 (Esc)', () => {
    closeSearchPanel(view);
    view.focus();
  });

  dom.append(input, count, prev, next, close);

  // 用当前查询在整个文档里数匹配数，并按光标位置算出当前第几个
  const refresh = () => {
    // 大小写不敏感：normalize 把两侧都转小写再比对
    const norm = (s: string) => s.toLowerCase();
    const total = (() => {
      if (!input.value) return 0;
      let n = 0;
      const cur = new SearchCursor(view.state.doc, norm(input.value), 0, view.state.doc.length, norm);
      while (!cur.next().done) n++;
      return n;
    })();
    let idx = 1;
    if (total > 0) {
      const pos = view.state.selection.main.head;
      const cur = new SearchCursor(view.state.doc, norm(input.value), 0, view.state.doc.length, norm);
      while (!cur.next().done) {
        // 光标落在该匹配内部或恰在匹配前 → 当前项是它
        if (cur.value.from >= pos) break;
        idx++;
      }
      if (idx > total) idx = 1;
    }
    count.textContent = total ? `${Math.min(idx, total)} / ${total}` : (input.value ? '无结果' : '');
  };

  input.oninput = () => {
    // 输入即时更新查询与高亮，但不跳转（与 Fyne 版交互约定一致：输入不抢焦点不跳）
    view.dispatch({effects: setSearchQuery.of(new SearchQuery({search: input.value, caseSensitive: false}))});
    refresh();
  };
  input.onkeydown = (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      e.shiftKey ? findPrevious(view) : findNext(view);
      refresh();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      closeSearchPanel(view);
      view.focus();
    }
  };

  return {
    dom,
    top: true,
    mount() {
      input.focus();
      input.select();
    },
    update() { refresh(); },
  };
}

export default function JsonEditor({value, onChange}: Props) {
  const hostRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  useEffect(() => {
    if (!hostRef.current) return;

    const view = new EditorView({
      parent: hostRef.current,
      state: EditorState.create({
        doc: value,
        extensions: [
          lineNumbers(),
          highlightActiveLineGutter(),
          highlightActiveLine(),
          history(),
          foldGutter(),              // 折叠箭头
          indentationMarkers(),      // 缩进参考线
          closeBrackets(),
          autocompletion(),
          search({top: true, createPanel: vscodeSearchPanel}),
          json(),
          syntaxHighlighting(jsonHighlight),
          EditorView.lineWrapping,
          EditorView.updateListener.of(u => {
            if (u.docChanged) onChange(u.state.doc.toString());
          }),
          EditorView.theme({
            '&': {backgroundColor: 'rgb(40, 43, 53)', color: 'rgb(210, 214, 226)', fontSize: '16px', height: '100%', position: 'relative'},
            '.cm-content': {fontFamily: '"Maple Mono", monospace', padding: '8px 0', caretColor: 'rgb(210, 214, 226)'},
            '.cm-gutters': {fontFamily: '"Maple Mono", monospace', backgroundColor: 'rgb(40, 43, 53)', color: 'rgb(121, 125, 134)', border: 'none'},
            '.cm-activeLine': {backgroundColor: 'rgba(210, 214, 226, 0.06)'},
            '.cm-activeLineGutter': {backgroundColor: 'rgba(210, 214, 226, 0.06)'},
            '.cm-foldGutter span': {cursor: 'pointer'},
            '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {backgroundColor: 'rgba(70, 75, 170, 0.4)'},
            '.cm-panels': {backgroundColor: 'transparent', color: 'rgb(210, 214, 226)'},
            '.cm-searchMatch': {backgroundColor: 'rgba(100, 230, 140, 0.25)'},
            '.cm-searchMatch-selected': {backgroundColor: 'rgba(100, 230, 140, 0.5)'},
            // 缩进参考线颜色
            '.cm-indentation-markers': {color: 'rgba(210, 214, 226, 0.15)'},
            // 搜索面板容器：贴在编辑器右上角
            '.cm-panels-top': {position: 'absolute', top: '8px', right: '16px', left: 'auto', zIndex: 10},
            // VS Code 风格搜索小窗
            '.cm-search-widget': {
              display: 'flex', alignItems: 'center', gap: '6px',
              padding: '6px 8px', borderRadius: '8px',
              background: 'rgb(28, 30, 38)', border: '1px solid rgb(70, 75, 170)',
              boxShadow: '0 4px 14px rgba(0,0,0,0.45)',
            },
            '.cm-search-widget input': {
              background: 'rgb(40, 43, 53)', color: 'rgb(210, 214, 226)',
              border: '1px solid rgb(121, 125, 134)', borderRadius: '6px',
              padding: '3px 8px', width: '170px', outline: 'none',
              fontFamily: '"Maple Mono", monospace', fontSize: '13px',
            },
            '.cm-search-widget button': {
              background: 'transparent', border: 'none', color: 'rgb(199, 202, 209)',
              cursor: 'pointer', fontSize: '13px', padding: '2px 6px', borderRadius: '4px',
            },
            '.cm-search-widget button:hover': {color: 'rgb(100, 230, 140)'},
            '.cm-search-count': {fontSize: '12px', color: 'rgb(121, 125, 134)', whiteSpace: 'nowrap'},
          }, {dark: true}),
          keymap.of([
            ...closeBracketsKeymap,
            ...defaultKeymap,
            ...searchKeymap,
            ...historyKeymap,
            ...completionKeymap,
            ...lintKeymap,
            indentWithTab,
          ]),
        ],
      }),
    });
    viewRef.current = view;
    return () => view.destroy();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 外部 value 变化（如格式化/压缩写回）时同步进编辑器
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (current !== value) {
      view.dispatch({changes: {from: 0, to: current.length, insert: value}});
    }
  }, [value]);

  return <div ref={hostRef} className="json-editor"/>;
}

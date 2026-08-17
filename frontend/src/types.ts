import type {ComponentType} from "react";

// Tool 接口契约（对应旧 Fyne 版 views.Tool）：每个工具页面内聚提供
// 标题、图标、工具栏右侧操作区（React 节点）、内容页。
// 新增工具 = pages/ 加一个页面 + tools 注册表加一行。
export interface Tool {
  id: string;
  title: string;
  icon: (selected: boolean) => string; // 图标 URL（选中白/未选中灰）
  toolbarActions: ComponentType;       // 工具栏右侧操作区
  content: ComponentType;              // 右侧中间的页面
}

import { CSSProperties, ReactNode } from 'react';
import { Link } from 'react-router-dom';
import ThemeToggle from '../ThemeToggle';

/**
 * 共享 PageHeader：统一标准页 header 结构，并在 actions 末尾自动追加 ThemeToggle。
 *
 * 设计取舍：
 *  - 不强制统一视觉；通过 `classes` 注入各页 module.css 类名映射，保留各页既有样式与
 *    动效，避免一次性改写 9 套 page header CSS 引入视觉回归。
 *  - actions slot 使用内联样式的 inline-flex 容器，避免污染每页 module.css 也避免
 *    不同页 header（grid / flex / space-between）布局机制冲突。
 *  - 视觉/对齐请由各页 module.css 自行维护；PageHeader 仅承担"骨架 + 行为契约"。
 */
export type BackConfig =
  | false
  | { to: string }
  | { onClick: () => void };

export interface PageHeaderClasses {
  header: string;
  backButton?: string;
  eyebrow?: string;
  /** 标题类名（如 ProductDetail 的 headerTitle、Home 的 title）；缺省则裸 h1 */
  title?: string;
}

interface PageHeaderProps {
  classes: PageHeaderClasses;
  back?: BackConfig;
  eyebrow?: string;
  title: ReactNode;
  /** 左侧自定义内容（如 Profile 头像）；不传则按 back -> title 顺序渲染 */
  leading?: ReactNode;
  /** 右侧 actions（除 ThemeToggle 外）；ThemeToggle 会在尾部自动追加 */
  actions?: ReactNode;
  /** 个别页面需要禁止 ThemeToggle（如 /login）时使用 */
  hideThemeToggle?: boolean;
}

const ACTIONS_STYLE: CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 8,
};

function PageHeader({
  classes,
  back = false,
  eyebrow,
  title,
  leading,
  actions,
  hideThemeToggle = false,
}: PageHeaderProps) {
  const renderBack = () => {
    if (back === false) return null;
    if ('to' in back) {
      return (
        <Link to={back.to} className={classes.backButton} aria-label="返回">
          ‹
        </Link>
      );
    }
    return (
      <button
        type="button"
        className={classes.backButton}
        onClick={back.onClick}
        aria-label="返回"
      >
        ‹
      </button>
    );
  };

  const renderTitleBlock = () => {
    if (eyebrow) {
      return (
        <div>
          <p className={classes.eyebrow}>{eyebrow}</p>
          <h1>{title}</h1>
        </div>
      );
    }
    if (classes.title) {
      return <h1 className={classes.title}>{title}</h1>;
    }
    return <h1>{title}</h1>;
  };

  return (
    <header className={classes.header}>
      {leading ?? renderBack()}
      {renderTitleBlock()}
      <div style={ACTIONS_STYLE}>
        {actions}
        {!hideThemeToggle && <ThemeToggle size="sm" />}
      </div>
    </header>
  );
}

export default PageHeader;

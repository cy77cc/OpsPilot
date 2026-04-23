import { createStyles } from 'antd-style';

export const useCopilotStyles = createStyles(({ token, css }) => ({
  surface: css`
    display: flex;
    flex-direction: column;
    height: 100%;
  `,
  header: css`
    display: flex;
    align-items: center;
  `,
  titleWrap: css`
    display: flex;
    flex-direction: column;
    gap: 4px;
  `,
  titleText: css`
    font-size: 18px;
    line-height: 26px;
    font-weight: 600;
    color: #111827;
  `,
  subtitleText: css`
    font-size: 13px;
    line-height: 20px;
    color: #6b7280;
  `,
  content: css`
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 16px;
    background: transparent;
  `,
  contentToolbar: css`
    display: flex;
    justify-content: flex-end;
    margin-bottom: 12px;
  `,
  headerActionBtn: css`
    width: 36px;
    height: 36px;
    font-size: 18px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  `,
  chatCard: css`
    background: transparent;
    border: none;
    border-radius: 0;
    padding: 0;
  `,
  senderWrap: css`
    padding: 12px 16px 16px;
  `,
  senderRow: css`
    display: flex;
    align-items: flex-end;
    gap: 6px;
  `,
  attachBtn: css`
    flex-shrink: 0;
    width: 36px;
    height: 36px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  `,
  senderFlex: css`
    flex: 1;
    min-width: 0;
  `,
  fileList: css`
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
  `,
  fileItem: css`
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 3px 6px 3px 8px;
    border-radius: 6px;
    background: ${token.colorFillSecondary};
    border: 1px solid ${token.colorBorderSecondary};
    font-size: 12px;
    color: ${token.colorText};
    max-width: 220px;
  `,
  fileName: css`
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
  `,
  emptyState: css`
    display: flex;
    flex-direction: column;
    gap: 16px;
  `,
  resizeHandle: css`
    position: absolute;
    top: 0;
    left: 0;
    width: 5px;
    height: 100%;
    cursor: col-resize;
    z-index: 100;
    background: transparent;
    border-left: 2px solid transparent;
    transition: border-color 0.15s;

    &:hover {
      border-left-color: ${token.colorPrimary};
    }
  `,
  scrollBottomBtn: css`
    && {
      position: absolute;
      left: 50%;
      bottom: 106px;
      transform: translateX(-50%);
      z-index: 120;
      width: 32px;
      min-width: 32px;
      max-width: 32px;
      height: 32px;
      padding: 0;
      border-radius: 999px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      color: #334155;
      box-shadow: 0 6px 14px rgba(15, 23, 42, 0.1);
      border: 1px solid rgba(203, 213, 225, 0.78);
      background: rgba(255, 255, 255, 0.94);
      backdrop-filter: blur(14px) saturate(1.15);
      -webkit-backdrop-filter: blur(14px) saturate(1.15);
      transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease, background 0.18s ease, color 0.18s ease;

      .anticon {
        font-size: 13px;
        color: inherit;
      }

      .anticon svg {
        color: inherit;
        fill: currentColor;
      }

      &&:hover {
        color: #0f172a;
        border-color: rgba(148, 163, 184, 0.95);
        background: rgba(255, 255, 255, 0.98);
        transform: translateX(-50%) translateY(-1px);
        box-shadow: 0 10px 18px rgba(15, 23, 42, 0.12);
        filter: none;
      }

      &&:focus,
      &&:focus-visible {
        color: #0f172a;
        border-color: rgba(148, 163, 184, 0.95);
        background: rgba(255, 255, 255, 0.98);
        box-shadow: 0 0 0 3px rgba(226, 232, 240, 0.85), 0 8px 16px rgba(15, 23, 42, 0.1);
        outline: none;
      }

      &:active {
        transform: translateX(-50%);
        box-shadow: 0 4px 10px rgba(15, 23, 42, 0.08);
      }
    }
  `,
  scrollBottomBtnLoading: css`
    position: relative;

    &::after {
      content: '';
      position: absolute;
      inset: -1px;
      border-radius: 999px;
      padding: 1px;
      background: linear-gradient(90deg, rgba(148, 163, 184, 0.08), rgba(59, 130, 246, 0.45), rgba(148, 163, 184, 0.08));
      background-size: 200% 100%;
      -webkit-mask:
        linear-gradient(#fff 0 0) content-box,
        linear-gradient(#fff 0 0);
      -webkit-mask-composite: xor;
      mask-composite: exclude;
      animation: scrollBtnSweep 1.4s linear infinite;
      pointer-events: none;
    }

    @keyframes scrollBtnSweep {
      from {
        background-position: 100% 50%;
      }
      to {
        background-position: -100% 50%;
      }
    }
  `,
}));

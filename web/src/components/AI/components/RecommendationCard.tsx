import React from 'react';
import { BulbOutlined } from '@ant-design/icons';
import { theme } from 'antd';
import type { EmbeddedRecommendation } from '../types';

interface RecommendationCardProps {
  recommendations: EmbeddedRecommendation[];
  onSelect: (prompt: string) => void;
}

/**
 * 推荐卡片组件
 * 展示 AI 生成的下一步建议
 */
export function RecommendationCard({ recommendations, onSelect }: RecommendationCardProps) {
  const { token } = theme.useToken();

  if (!recommendations || recommendations.length === 0) {
    return null;
  }

  return (
    <div style={{
      marginTop: 16,
      padding: 12,
      background: token.colorPrimaryBg,
      borderRadius: token.borderRadius,
      border: `1px solid ${token.colorPrimaryBorder}`,
    }}>
      {/* 标题 */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: 6,
        marginBottom: 12,
        color: token.colorPrimary,
        fontWeight: 500,
        fontSize: 13,
      }}>
        <BulbOutlined />
        <span>下一步建议</span>
      </div>

      {/* 推荐列表 */}
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}>
        {recommendations.slice(0, 3).map((rec, index) => (
          <div
            key={rec.id || index}
            onClick={() => rec.followup_prompt && onSelect(rec.followup_prompt)}
            style={{
              padding: '10px 12px',
              background: token.colorBgContainer,
              borderRadius: token.borderRadiusSM,
              cursor: rec.followup_prompt ? 'pointer' : 'default',
              border: `1px solid ${token.colorBorderSecondary}`,
              transition: 'all 0.2s ease',
              ...(rec.followup_prompt ? {
                ':hover': {
                  borderColor: token.colorPrimary,
                  background: token.colorPrimaryBg,
                },
              } : {}),
            }}
            onMouseEnter={(e) => {
              if (rec.followup_prompt) {
                e.currentTarget.style.borderColor = token.colorPrimary;
                e.currentTarget.style.background = token.colorPrimaryBg;
              }
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = token.colorBorderSecondary;
              e.currentTarget.style.background = token.colorBgContainer;
            }}
          >
            <div style={{
              fontWeight: 500,
              fontSize: 13,
              color: token.colorText,
              marginBottom: 4,
            }}>
              {rec.title}
            </div>
            {rec.content && (
              <div style={{
                fontSize: 12,
                color: token.colorTextSecondary,
                lineHeight: 1.5,
              }}>
                {rec.content}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

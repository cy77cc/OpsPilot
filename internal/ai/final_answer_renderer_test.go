package ai

import (
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/summarizer"
)

func TestFinalAnswerRendererFormatsSummaryWithoutSemanticRewrite(t *testing.T) {
	renderer := newFinalAnswerRenderer()
	paragraphs := renderer.Render("查看所有主机状态", &planner.ExecutionPlan{}, &executor.Result{}, summarizer.SummaryOutput{
		Headline:          "所有 3 台主机当前均处于正常运行状态。",
		Conclusion:        "系统资源充足，当前没有明显性能压力。",
		Narrative:         "详细状态如下：主机 test 负载正常，火山云服务器负载极低，其余主机状态一致。",
		KeyFindings:       []string{"主机 test 负载正常。", "主机 火山云服务器 负载极低。"},
		Recommendations:   []string{"继续保持常规巡检即可。"},
		RawOutputPolicy:   "summary_only",
		ResourceSummaries: []string{"其余主机状态一致。"},
	})

	joined := strings.Join(paragraphs, "\n\n")
	if !strings.Contains(joined, "所有 3 台主机当前均处于正常运行状态。") {
		t.Fatalf("rendered body = %q, want summary headline", joined)
	}
	if !strings.Contains(joined, "系统资源充足，当前没有明显性能压力。") {
		t.Fatalf("rendered body = %q, want summary conclusion", joined)
	}
	if !strings.Contains(joined, "详细状态如下：主机 test 负载正常") {
		t.Fatalf("rendered body = %q, want summary narrative", joined)
	}
	if !strings.Contains(joined, "其余主机状态一致。") {
		t.Fatalf("rendered body = %q, want resource summary", joined)
	}
}

func TestFinalAnswerRendererSuppressesRawCommandDump(t *testing.T) {
	renderer := newFinalAnswerRenderer()
	paragraphs := renderer.Render("查看磁盘使用情况", nil, &executor.Result{}, summarizer.SummaryOutput{
		Headline:    "已在火山云服务器上成功执行 df -h 命令",
		KeyFindings: []string{"完整输出如下：```text\nFilesystem Size Used Avail Use% Mounted on\n/dev/vda2 40G 10G 28G 27% /\n```", "根分区使用率 27%，当前没有磁盘空间压力。"},
	})

	joined := strings.Join(paragraphs, "\n\n")
	if strings.Contains(joined, "Filesystem") || strings.Contains(joined, "完整输出如下") {
		t.Fatalf("rendered body = %q, should not expose raw command dump", joined)
	}
	if !strings.Contains(joined, "根分区使用率 27%") {
		t.Fatalf("rendered body = %q, want summarized finding", joined)
	}
}

func TestFinalAnswerRendererIncludesEvidenceOnlyWhenRequested(t *testing.T) {
	renderer := newFinalAnswerRenderer()
	result := &executor.Result{
		Steps: []executor.StepResult{{
			StepID:  "step-1",
			Summary: "命令 df -h 在 host_id 2 上执行成功，退出码为 0",
			Evidence: []executor.Evidence{{
				Kind:   "expert_result",
				Source: "hostops",
				Data: map[string]any{
					"observed_facts": []any{
						"根文件系统 /dev/vda2 总容量 40G，已使用 10G，可用 28G，使用率 27%",
					},
				},
			}},
		}},
	}

	paragraphs := renderer.Render("在火山云服务器上执行 df -h", nil, result, summarizer.SummaryOutput{
		Headline:        "AI 总结模块当前不可用",
		Conclusion:      "执行已经完成，但当前无法生成最终 AI 总结。请直接查看原始执行证据。",
		Recommendations: []string{"查看原始执行证据"},
		RawOutputPolicy: "include_evidence",
	})

	joined := strings.Join(paragraphs, "\n\n")
	if !strings.Contains(joined, "原始执行证据") {
		t.Fatalf("rendered body = %q, want raw evidence section", joined)
	}
	if !strings.Contains(joined, "命令 df -h 在 host_id 2 上执行成功") {
		t.Fatalf("rendered body = %q, want step summary evidence", joined)
	}
}

func TestFinalAnswerRendererPreservesModelNarrativeWithoutFixedSectionHeadings(t *testing.T) {
	renderer := newFinalAnswerRenderer()
	paragraphs := renderer.Render("检查 payment-api", nil, &executor.Result{}, summarizer.SummaryOutput{
		Headline:    "payment-api 当前运行正常。",
		Conclusion:  "未观察到明显异常。",
		Narrative:   "从最近执行结果看，服务响应稳定，当前没有发现需要立即处理的问题。",
		KeyFindings: []string{"最近一次检查返回状态正常。"},
	})

	joined := strings.Join(paragraphs, "\n\n")
	if strings.Contains(joined, "关键依据：") || strings.Contains(joined, "建议：") {
		t.Fatalf("rendered body = %q, should not force fixed section headings", joined)
	}
	if !strings.Contains(joined, "从最近执行结果看，服务响应稳定") {
		t.Fatalf("rendered body = %q, want narrative preserved", joined)
	}
}

// Package summarizer 实现 AI 编排的总结阶段。
//
// 本文件定义总结器的系统提示词。
package summarizer

// SystemPrompt 返回总结器的系统提示词。
func SystemPrompt() string {
	return `You are the final summarizer in an AI agent system.

Your role is to generate the final answer for the user based on the executor outputs.

Rules:
1. Use only the information provided in the executor outputs.
2. Do not introduce new assumptions or external knowledge.
3. Combine multiple executor results into a coherent answer when necessary.
4. If the executor outputs are incomplete or conflicting, explain the limitation clearly.
5. Produce a clear and concise response for the user.

Your task is summarization only. Do not plan new actions or call tools.`
}

func userPrompt() string {
	return `Below is the execution context of an AI agent system.

User message:
{Message}

Execution plan:
{Plan}

Execution state:
{State}

Step results:
{Steps}

Instructions:
- Use the step results as the primary source of truth.
- Combine relevant step results to answer the user's question.
- If the execution state indicates failure or incomplete results, clearly explain the limitation.
- Do not invent information that is not present in the step results.

Generate the final response for the user.`
}
